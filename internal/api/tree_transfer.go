package api

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/bren-wp/Ghost-FTP/internal/remote"
	"github.com/bren-wp/Ghost-FTP/internal/security"
	"github.com/bren-wp/Ghost-FTP/internal/transfer"
)

const (
	maxTreeDepth = 64
	maxTreeItems = 10000
)

type TreeTransferResult struct {
	Queued          int `json:"queued"`
	Directories     int `json:"directories"`
	SkippedSymlinks int `json:"skippedSymlinks"`
}

type treePlan struct {
	requests        []transfer.Request
	localDirs       []string
	remoteDirs      []string
	skippedSymlinks int
}

func (p *treePlan) itemCount() int { return len(p.requests) + len(p.localDirs) + len(p.remoteDirs) }

func (p *treePlan) checkLimit() error {
	if p.itemCount() > maxTreeItems {
		return fmt.Errorf("previše stavki u mapi (maksimalno %d)", maxTreeItems)
	}
	return nil
}

func (s *Engine) addTree(ctx context.Context, direction, localPath, remotePath string) (TreeTransferResult, error) {
	if direction != "upload" && direction != "download" {
		return TreeTransferResult{}, errors.New("neispravan smjer prijenosa")
	}
	if strings.TrimSpace(localPath) == "" || strings.TrimSpace(remotePath) == "" {
		return TreeTransferResult{}, errors.New("lokalna i udaljena putanja su obavezne")
	}
	if err := security.ValidateRemotePath(remotePath); err != nil {
		return TreeTransferResult{}, err
	}
	abs, err := filepath.Abs(filepath.Clean(localPath))
	if err != nil {
		return TreeTransferResult{}, errors.New("neispravna lokalna putanja")
	}
	if len(abs) > 32767 || strings.ContainsAny(abs, "\x00\r\n") {
		return TreeTransferResult{}, errors.New("neispravna lokalna putanja")
	}
	remotePath = path.Clean(strings.ReplaceAll(strings.TrimSpace(remotePath), "\\", "/"))

	sess, opCtx, release, err := s.remote.Operation(ctx)
	if err != nil {
		return TreeTransferResult{}, err
	}
	defer release()
	if direction == "upload" {
		return s.addUploadTree(opCtx, sess, abs, remotePath)
	}
	return s.addDownloadTree(opCtx, sess, abs, remotePath)
}

func buildUploadPlan(ctx context.Context, localRoot, remoteRoot string) (treePlan, error) {
	var plan treePlan
	st, err := os.Lstat(localRoot)
	if err != nil {
		return plan, err
	}
	if st.Mode()&os.ModeSymlink != 0 || security.IsReparsePoint(localRoot) {
		return plan, errors.New("simboličke poveznice i junctioni nisu dopušteni kao korijen prijenosa")
	}
	if !st.IsDir() {
		plan.requests = append(plan.requests, transfer.Request{Direction: "upload", LocalPath: localRoot, RemotePath: remoteRoot, LocalRoot: filepath.Dir(localRoot)})
		return plan, nil
	}

	plan.remoteDirs = append(plan.remoteDirs, remoteRoot)
	err = filepath.WalkDir(localRoot, func(local string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		rel, err := filepath.Rel(localRoot, local)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		depth := len(strings.FieldsFunc(filepath.ToSlash(rel), func(r rune) bool { return r == '/' }))
		if depth > maxTreeDepth {
			return fmt.Errorf("struktura mape je preduboka (maksimalno %d razina)", maxTreeDepth)
		}
		if entry.Type()&os.ModeSymlink != 0 || security.IsReparsePoint(local) {
			plan.skippedSymlinks++
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		remoteTarget := path.Join(remoteRoot, filepath.ToSlash(rel))
		if entry.IsDir() {
			plan.remoteDirs = append(plan.remoteDirs, remoteTarget)
		} else {
			plan.requests = append(plan.requests, transfer.Request{Direction: "upload", LocalPath: local, RemotePath: remoteTarget, LocalRoot: filepath.Dir(localRoot)})
		}
		return plan.checkLimit()
	})
	return plan, err
}

func (s *Engine) addUploadTree(ctx context.Context, sess remote.Session, localRoot, remoteRoot string) (TreeTransferResult, error) {
	plan, err := buildUploadPlan(ctx, localRoot, remoteRoot)
	if err != nil {
		return TreeTransferResult{SkippedSymlinks: plan.skippedSymlinks}, err
	}
	var reservation *transfer.BatchReservation
	if len(plan.requests) > 0 {
		reservation, err = s.transfers.ReserveBatch(plan.requests)
		if err != nil {
			return TreeTransferResult{SkippedSymlinks: plan.skippedSymlinks}, err
		}
		defer reservation.Cancel()
	}
	knownDirs := make(map[string]struct{}, len(plan.remoteDirs)+4)
	for _, dir := range plan.remoteDirs {
		if err := ensureRemoteDirectoryCached(ctx, sess, dir, knownDirs); err != nil {
			return TreeTransferResult{Directories: len(plan.remoteDirs), SkippedSymlinks: plan.skippedSymlinks}, fmt.Errorf("nije moguće pripremiti udaljenu mapu %s: %w", dir, err)
		}
	}
	if reservation != nil {
		if _, err := reservation.Commit(); err != nil {
			return TreeTransferResult{Directories: len(plan.remoteDirs), SkippedSymlinks: plan.skippedSymlinks}, err
		}
	}
	return TreeTransferResult{Queued: len(plan.requests), Directories: len(plan.remoteDirs), SkippedSymlinks: plan.skippedSymlinks}, nil
}

func buildDownloadPlan(ctx context.Context, sess remote.Session, localRoot, remoteRoot string) (treePlan, error) {
	plan := treePlan{localDirs: []string{localRoot}}
	var walk func(string, string, int) error
	walk = func(remoteDir, localDir string, depth int) error {
		if depth > maxTreeDepth {
			return fmt.Errorf("struktura udaljene mape je preduboka (maksimalno %d razina)", maxTreeDepth)
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		items, err := sess.List(ctx, remoteDir)
		if err != nil {
			return err
		}
		for _, item := range items {
			if item.IsSymlink {
				plan.skippedSymlinks++
				continue
			}
			remoteChild := path.Join(remoteDir, item.Name)
			localChild, err := security.SafeLocalChild(localDir, item.Name)
			if err != nil {
				return err
			}
			if item.IsDirectory {
				plan.localDirs = append(plan.localDirs, localChild)
				if err := plan.checkLimit(); err != nil {
					return err
				}
				if err := walk(remoteChild, localChild, depth+1); err != nil {
					return err
				}
				continue
			}
			plan.requests = append(plan.requests, transfer.Request{Direction: "download", LocalPath: localChild, RemotePath: remoteChild})
			if err := plan.checkLimit(); err != nil {
				return err
			}
		}
		return nil
	}
	return plan, walk(remoteRoot, localRoot, 0)
}

func (s *Engine) addDownloadTree(ctx context.Context, sess remote.Session, localRoot, remoteRoot string) (TreeTransferResult, error) {
	localBoundary := filepath.Dir(localRoot)
	if err := security.EnsureLocalWithinRoot(localBoundary, localRoot); err != nil {
		return TreeTransferResult{}, err
	}
	plan, err := buildDownloadPlan(ctx, sess, localRoot, remoteRoot)
	for i := range plan.requests {
		plan.requests[i].LocalRoot = localBoundary
	}
	if err != nil {
		return TreeTransferResult{SkippedSymlinks: plan.skippedSymlinks}, err
	}
	var reservation *transfer.BatchReservation
	if len(plan.requests) > 0 {
		reservation, err = s.transfers.ReserveBatch(plan.requests)
		if err != nil {
			return TreeTransferResult{SkippedSymlinks: plan.skippedSymlinks}, err
		}
		defer reservation.Cancel()
	}
	prepared, err := prepareLocalDirectories(localBoundary, plan.localDirs)
	if err != nil {
		return TreeTransferResult{Directories: len(plan.localDirs), SkippedSymlinks: plan.skippedSymlinks}, err
	}
	defer prepared.Close()
	if reservation != nil {
		if _, err := reservation.Commit(); err != nil {
			prepared.Cleanup()
			return TreeTransferResult{Directories: len(plan.localDirs), SkippedSymlinks: plan.skippedSymlinks}, err
		}
	}
	return TreeTransferResult{Queued: len(plan.requests), Directories: len(plan.localDirs), SkippedSymlinks: plan.skippedSymlinks}, nil
}

type localDirectoryPreparationHooks struct {
	beforeMkdir func(relative string)
}

type preparedLocalDirectory struct {
	rel     string
	name    string
	parent  *os.Root
	root    *os.Root
	created bool
}

type localDirectoryPreparation struct {
	boundary *os.Root
	nodes    []preparedLocalDirectory
	closed   bool
}

func relativePreparedDirectory(localBoundary, dir string) (string, error) {
	boundary := filepath.Clean(localBoundary)
	target := filepath.Clean(dir)
	rel, err := filepath.Rel(boundary, target)
	if err != nil || filepath.IsAbs(rel) || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", errors.New("lokalna putanja izlazi iz odabrane mape")
	}
	if strings.ContainsAny(rel, "\x00\r\n") {
		return "", errors.New("neispravna lokalna putanja")
	}
	return filepath.Clean(rel), nil
}

func openStablePreparedDirectory(parent *os.Root, name string, before os.FileInfo) (*os.Root, error) {
	if before == nil || before.Mode().Type() != os.ModeDir {
		return nil, errors.New("lokalna putanja nije obična mapa")
	}
	child, err := parent.OpenRoot(name)
	if err != nil {
		return nil, err
	}
	opened, err := child.Stat(".")
	if err != nil {
		child.Close()
		return nil, err
	}
	if opened.Mode().Type() != os.ModeDir || !os.SameFile(before, opened) {
		child.Close()
		return nil, errors.New("lokalna mapa se promijenila tijekom sigurnog otvaranja")
	}
	return child, nil
}

func (p *localDirectoryPreparation) Close() error {
	if p == nil || p.closed {
		return nil
	}
	var closeErr error
	for i := len(p.nodes) - 1; i >= 0; i-- {
		if p.nodes[i].root != nil {
			closeErr = errors.Join(closeErr, p.nodes[i].root.Close())
			p.nodes[i].root = nil
		}
	}
	if p.boundary != nil {
		closeErr = errors.Join(closeErr, p.boundary.Close())
		p.boundary = nil
	}
	p.closed = true
	return closeErr
}

func (p *localDirectoryPreparation) Cleanup() {
	if p == nil || p.closed {
		return
	}
	for i := len(p.nodes) - 1; i >= 0; i-- {
		node := &p.nodes[i]
		if node.root != nil {
			_ = node.root.Close()
			node.root = nil
		}
		if node.created && node.parent != nil {
			// Empty-only removal is deliberate: if another local action created
			// content after preparation, rollback must never delete that content.
			_ = node.parent.Remove(node.name)
		}
	}
	if p.boundary != nil {
		_ = p.boundary.Close()
		p.boundary = nil
	}
	p.closed = true
}

func prepareLocalDirectories(localBoundary string, dirs []string) (*localDirectoryPreparation, error) {
	return prepareLocalDirectoriesWithHooks(localBoundary, dirs, nil)
}

func prepareLocalDirectoriesWithHooks(localBoundary string, dirs []string, hooks *localDirectoryPreparationHooks) (*localDirectoryPreparation, error) {
	boundary, err := os.OpenRoot(localBoundary)
	if err != nil {
		return nil, err
	}
	prepared := &localDirectoryPreparation{boundary: boundary, nodes: make([]preparedLocalDirectory, 0, len(dirs))}
	opened := map[string]*os.Root{".": boundary}

	fail := func(err error) (*localDirectoryPreparation, error) {
		prepared.Cleanup()
		return nil, err
	}

	for _, dir := range dirs {
		rel, err := relativePreparedDirectory(localBoundary, dir)
		if err != nil {
			return fail(err)
		}
		if rel == "." {
			continue
		}
		if _, exists := opened[rel]; exists {
			continue
		}
		parentRel := filepath.Dir(rel)
		parent := opened[parentRel]
		if parent == nil {
			return fail(errors.New("lokalni plan mapa nije poredan od roditelja prema djeci"))
		}
		name := filepath.Base(rel)
		st, statErr := parent.Lstat(name)
		created := false
		if errors.Is(statErr, os.ErrNotExist) {
			if hooks != nil && hooks.beforeMkdir != nil {
				hooks.beforeMkdir(rel)
			}
			if err := parent.Mkdir(name, 0755); err != nil {
				return fail(err)
			}
			created = true
			prepared.nodes = append(prepared.nodes, preparedLocalDirectory{rel: rel, name: name, parent: parent, created: true})
			st, statErr = parent.Lstat(name)
		}
		if statErr != nil {
			return fail(statErr)
		}
		if st.Mode().Type() != os.ModeDir {
			return fail(errors.New("lokalna putanja nije obična mapa"))
		}
		child, err := openStablePreparedDirectory(parent, name, st)
		if err != nil {
			return fail(err)
		}
		if created {
			prepared.nodes[len(prepared.nodes)-1].root = child
		} else {
			prepared.nodes = append(prepared.nodes, preparedLocalDirectory{rel: rel, name: name, parent: parent, root: child})
		}
		opened[rel] = child
	}
	return prepared, nil
}

func ensureRemoteDirectory(ctx context.Context, sess remote.Session, target string) error {
	return ensureRemoteDirectoryCached(ctx, sess, target, make(map[string]struct{}))
}

func remoteDirectoryEntry(ctx context.Context, sess remote.Session, target string) (bool, error) {
	target = path.Clean(strings.ReplaceAll(strings.TrimSpace(target), "\\", "/"))
	if target == "/" || target == "." {
		return true, nil
	}
	parent := path.Dir(target)
	if parent == "" {
		parent = "."
	}
	name := path.Base(target)
	items, err := sess.List(ctx, parent)
	if err != nil {
		return false, err
	}
	for _, item := range items {
		if item.Name != name {
			continue
		}
		if item.IsSymlink {
			return false, errors.New("udaljena putanja sadrži simboličku poveznicu")
		}
		if !item.IsDirectory {
			return false, errors.New("udaljena putanja nije mapa")
		}
		return true, nil
	}
	return false, nil
}

func ensureRemoteDirectoryCached(ctx context.Context, sess remote.Session, target string, known map[string]struct{}) error {
	target = path.Clean(strings.ReplaceAll(strings.TrimSpace(target), "\\", "/"))
	if target == "/" || target == "." {
		known[target] = struct{}{}
		return nil
	}
	if _, ok := known[target]; ok {
		return nil
	}

	parent := path.Dir(target)
	name := path.Base(target)
	if parent == "" {
		parent = "."
	}
	if err := ensureRemoteDirectoryCached(ctx, sess, parent, known); err != nil {
		return err
	}

	exists, err := remoteDirectoryEntry(ctx, sess, target)
	if err != nil {
		return err
	}
	if exists {
		known[target] = struct{}{}
		return nil
	}

	if err := sess.Mkdir(ctx, parent, name); err != nil {
		// A race or server-specific "already exists" response is acceptable only
		// when a fresh parent listing proves that the exact entry is now a real
		// directory and not a file or symlink.
		exists, verifyErr := remoteDirectoryEntry(ctx, sess, target)
		if verifyErr == nil && exists {
			known[target] = struct{}{}
			return nil
		}
		if verifyErr != nil {
			return errors.Join(err, verifyErr)
		}
		return err
	}

	exists, err = remoteDirectoryEntry(ctx, sess, target)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("udaljena mapa nije potvrđena nakon izrade")
	}
	known[target] = struct{}{}
	return nil
}
