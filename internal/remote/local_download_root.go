package remote

import (
	"bytes"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/bren-wp/Ghost-FTP/internal/security"
)

const localDownloadSentinelSize = 32

type localDownloadTarget struct {
	root       *os.Root
	rootPath   string
	targetPath string
	targetRel  string
	partName   string
	partPath   string
	partInfo   os.FileInfo
	sentinel   []byte
}

func localPathRelativeToRoot(rootPath, targetPath string) (string, error) {
	rootAbs, err := filepath.Abs(filepath.Clean(rootPath))
	if err != nil {
		return "", err
	}
	targetAbs, err := filepath.Abs(filepath.Clean(targetPath))
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(rootAbs, targetAbs)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", errors.New("lokalna datoteka nije unutar dopuštenog korijena preuzimanja")
	}
	return filepath.Clean(rel), nil
}

func prepareLocalDownloadTarget(local, requestedRoot string, skipExisting bool) (*localDownloadTarget, error) {
	local = filepath.Clean(local)
	if requestedRoot == "" {
		requestedRoot = filepath.Dir(local)
	}
	rootPath, err := filepath.Abs(filepath.Clean(requestedRoot))
	if err != nil {
		return nil, err
	}
	targetPath, err := filepath.Abs(local)
	if err != nil {
		return nil, err
	}
	if err := security.EnsureLocalWithinRoot(rootPath, targetPath); err != nil {
		return nil, err
	}
	targetRel, err := localPathRelativeToRoot(rootPath, targetPath)
	if err != nil {
		return nil, err
	}

	root, err := os.OpenRoot(rootPath)
	if err != nil {
		return nil, err
	}
	cleanupRoot := true
	defer func() {
		if cleanupRoot {
			_ = root.Close()
		}
	}()

	parentRel := filepath.Dir(targetRel)
	if parentRel != "." {
		if err := root.MkdirAll(parentRel, 0755); err != nil {
			return nil, err
		}
	}
	// Preserve Ghost FTP's stricter policy of rejecting links/junctions below the
	// user-selected root. The Root handle remains the containment boundary even if
	// the filesystem changes again immediately after this policy check.
	if err := security.EnsureLocalWithinRoot(rootPath, targetPath); err != nil {
		return nil, err
	}

	if st, err := root.Lstat(targetRel); err == nil {
		if !st.Mode().IsRegular() || st.Mode()&os.ModeSymlink != 0 || security.IsReparsePoint(targetPath) {
			return nil, errors.New("ciljna putanja nije obična datoteka")
		}
		if skipExisting {
			return nil, ErrSkipped
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	rootReal := rootPath
	if resolved, err := filepath.EvalSymlinks(rootPath); err == nil {
		if abs, absErr := filepath.Abs(resolved); absErr == nil {
			rootReal = abs
		}
	}

	token, err := randomTransferToken()
	if err != nil {
		return nil, err
	}
	partName := ".GhostFTP-part-" + token
	part, err := root.OpenFile(partName, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return nil, err
	}
	sentinel := make([]byte, localDownloadSentinelSize)
	if _, err = rand.Read(sentinel); err != nil {
		_ = part.Close()
		_ = root.Remove(partName)
		return nil, err
	}
	if _, err = part.Write(sentinel); err != nil {
		_ = part.Close()
		_ = root.Remove(partName)
		return nil, err
	}
	if err = part.Sync(); err != nil {
		_ = part.Close()
		_ = root.Remove(partName)
		return nil, err
	}
	partInfo, err := part.Stat()
	if closeErr := part.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		_ = root.Remove(partName)
		return nil, err
	}

	cleanupRoot = false
	return &localDownloadTarget{
		root: root, rootPath: rootPath, targetPath: targetPath, targetRel: targetRel,
		partName: partName, partPath: filepath.Join(rootReal, partName), partInfo: partInfo, sentinel: sentinel,
	}, nil
}

func (d *localDownloadTarget) Close() error {
	if d == nil || d.root == nil {
		return nil
	}
	err := d.root.Close()
	d.root = nil
	return err
}

func (d *localDownloadTarget) cleanupPart() {
	if d == nil || d.root == nil || d.partName == "" {
		return
	}
	_ = d.root.Remove(d.partName)
}

func (d *localDownloadTarget) validatePart() error {
	if d == nil || d.root == nil {
		return errors.New("lokalni korijen preuzimanja nije dostupan")
	}
	st, err := d.root.Lstat(d.partName)
	if err != nil {
		return err
	}
	if !st.Mode().IsRegular() || st.Mode()&os.ModeSymlink != 0 || !os.SameFile(d.partInfo, st) || security.IsReparsePoint(d.partPath) {
		return errors.New("preuzeta privremena datoteka nije ista obična lokalna datoteka koja je sigurno pripremljena")
	}
	f, err := d.root.Open(d.partName)
	if err != nil {
		return err
	}
	prefix := make([]byte, len(d.sentinel))
	n, readErr := io.ReadFull(f, prefix)
	closeErr := f.Close()
	if readErr != nil && !errors.Is(readErr, io.EOF) && !errors.Is(readErr, io.ErrUnexpectedEOF) {
		return readErr
	}
	if closeErr != nil {
		return closeErr
	}
	if n == len(d.sentinel) && bytes.Equal(prefix, d.sentinel) {
		return errors.New("alat za prijenos nije zamijenio sigurno pripremljenu privremenu datoteku")
	}
	return nil
}

func rootSiblingRelative(targetPath, rootPath, kind string, preserveBase bool) (string, error) {
	candidate, err := localTransferSibling(targetPath, kind, preserveBase)
	if err != nil {
		return "", err
	}
	return localPathRelativeToRoot(rootPath, candidate)
}

func rootEntryIsSame(root *os.Root, name string, expected os.FileInfo) bool {
	if root == nil || expected == nil {
		return false
	}
	st, err := root.Lstat(name)
	return err == nil && os.SameFile(expected, st)
}

func restoreRootBackup(root *os.Root, backupRel, targetRel string) error {
	if root == nil || backupRel == "" {
		return nil
	}
	if _, err := root.Lstat(targetRel); err == nil {
		return errors.New("ciljna datoteka se ponovno pojavila; sigurnosna kopija nije prepisana")
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	// Prefer a hard-link restore because creation of targetRel is no-replace.
	// If the filesystem does not support links, fall back to root-relative Rename;
	// the preceding absence check is advisory, while os.Root still guarantees the
	// operation cannot escape the selected root.
	if err := root.Link(backupRel, targetRel); err == nil {
		if removeErr := root.Remove(backupRel); removeErr != nil {
			return fmt.Errorf("izvorna datoteka je vraćena, ali rollback kopija nije uklonjena: %w", removeErr)
		}
		return nil
	}
	return root.Rename(backupRel, targetRel)
}

func (d *localDownloadTarget) activate(keepBackup, skipExisting bool) error {
	if err := d.validatePart(); err != nil {
		d.cleanupPart()
		return err
	}
	if err := security.EnsureLocalWithinRoot(d.rootPath, d.targetPath); err != nil {
		d.cleanupPart()
		return err
	}

	var (
		existed    bool
		backupRel  string
		backupInfo os.FileInfo
	)
	if st, err := d.root.Lstat(d.targetRel); err == nil {
		if !st.Mode().IsRegular() || st.Mode()&os.ModeSymlink != 0 || security.IsReparsePoint(d.targetPath) {
			d.cleanupPart()
			return errors.New("ciljna putanja nije obična datoteka")
		}
		if skipExisting {
			d.cleanupPart()
			return ErrSkipped
		}
		existed = true
		kind := "rollback"
		preserveBase := false
		if keepBackup {
			kind = "backup"
			preserveBase = true
		}
		backupRel, err = rootSiblingRelative(d.targetPath, d.rootPath, kind, preserveBase)
		if err != nil {
			d.cleanupPart()
			return err
		}
		if _, err := d.root.Lstat(backupRel); err == nil {
			d.cleanupPart()
			return errors.New("nasumična lokalna sigurnosna kopija već postoji")
		} else if !errors.Is(err, os.ErrNotExist) {
			d.cleanupPart()
			return err
		}
		if err := d.root.Rename(d.targetRel, backupRel); err != nil {
			d.cleanupPart()
			return err
		}
		backupInfo, err = d.root.Lstat(backupRel)
		if err != nil || !backupInfo.Mode().IsRegular() {
			d.cleanupPart()
			return errors.New("nije moguće potvrditi lokalnu sigurnosnu kopiju prije aktivacije")
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		d.cleanupPart()
		return err
	}

	placeholder, err := d.root.OpenFile(d.targetRel, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		d.cleanupPart()
		if existed {
			return fmt.Errorf("ciljna datoteka se promijenila prije aktivacije; izvorna datoteka ostaje u sigurnosnoj kopiji %s: %w", backupRel, err)
		}
		return err
	}
	placeholderInfo, statErr := placeholder.Stat()
	closeErr := placeholder.Close()
	if statErr != nil || closeErr != nil {
		if rootEntryIsSame(d.root, d.targetRel, placeholderInfo) {
			_ = d.root.Remove(d.targetRel)
		}
		d.cleanupPart()
		if existed {
			_ = restoreRootBackup(d.root, backupRel, d.targetRel)
		}
		return errors.Join(statErr, closeErr)
	}
	if !rootEntryIsSame(d.root, d.targetRel, placeholderInfo) {
		d.cleanupPart()
		if existed {
			_ = restoreRootBackup(d.root, backupRel, d.targetRel)
		}
		return errors.New("ciljna datoteka se promijenila neposredno prije aktivacije")
	}

	if err := d.root.Rename(d.partName, d.targetRel); err != nil {
		if rootEntryIsSame(d.root, d.targetRel, placeholderInfo) {
			_ = d.root.Remove(d.targetRel)
		}
		d.cleanupPart()
		if existed {
			if restoreErr := restoreRootBackup(d.root, backupRel, d.targetRel); restoreErr != nil {
				return fmt.Errorf("aktivacija nove datoteke nije uspjela, a izvorna datoteka ostaje u sigurnosnoj kopiji %s: %w", backupRel, restoreErr)
			}
		}
		return err
	}

	if existed && !keepBackup {
		if !rootEntryIsSame(d.root, backupRel, backupInfo) {
			return fmt.Errorf("nova datoteka je aktivna, ali rollback kopija %s promijenila se prije čišćenja", backupRel)
		}
		if err := d.root.Remove(backupRel); err != nil {
			return fmt.Errorf("nova datoteka je aktivna, ali lokalni rollback nije moguće ukloniti na %s: %w", backupRel, err)
		}
	}
	return nil
}
