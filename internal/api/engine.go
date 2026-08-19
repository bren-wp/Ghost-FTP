package api

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"brendigo.com/byftp/internal/brand"
	"brendigo.com/byftp/internal/config"
	"brendigo.com/byftp/internal/localfs"
	"brendigo.com/byftp/internal/model"
	"brendigo.com/byftp/internal/platform"
	"brendigo.com/byftp/internal/remote"
	"brendigo.com/byftp/internal/security"
	"brendigo.com/byftp/internal/transfer"
)

type Engine struct {
	dataDir   string
	profiles  *config.Profiles
	settings  *config.SettingsStore
	local     *localfs.Service
	remote    *remote.Manager
	transfers *transfer.Manager
	closeOnce sync.Once
}

func cleanupLegacyDiagnostics(dataDir string) {
	for _, name := range []string{"byftp.log", "byftp.log.1"} {
		p := filepath.Join(dataDir, name)
		if _, err := os.Lstat(p); err == nil {
			// Remove only the directory entry. os.Remove never follows a symlink.
			_ = os.Remove(p)
		}
	}
}

func New(dataDir, exePath string) (*Engine, error) {
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return nil, err
	}
	cleanupLegacyDiagnostics(dataDir)
	store := config.New(dataDir)
	p := config.NewProfiles(store)
	ss := config.NewSettings(store)
	r := remote.NewManager(p, ss, dataDir, exePath)
	e := &Engine{dataDir: dataDir, profiles: p, settings: ss, local: localfs.New(), remote: r}
	e.transfers = transfer.New(r, ss)
	return e, nil
}

// recoverTypedCall converts an unexpected panic at a typed UI/engine boundary
// into one stable user-safe error. Keeping this logic in one helper prevents
// individual secret-bearing entry points from drifting to different recovery
// behavior while still allowing lower layers to use ordinary Go errors.
func recoverTypedCall(callErr *error) {
	if recover() != nil {
		*callErr = errors.New("interna greška aplikacije")
	}
}

// The desktop UI and engine live in the same process. All public operations are
// typed calls: there is no JSON dispatcher, localhost server, browser IPC or
// generic string channel between the UI and the transfer engine.
func (e *Engine) ChooseDirectory() (string, error)         { return platform.ChooseDirectory() }
func (e *Engine) ChoosePrivateKey() (string, error)        { return platform.ChoosePrivateKey() }
func (e *Engine) Profiles() ([]model.PublicProfile, error) { return e.profiles.List() }
func (e *Engine) RemoveProfile(id string) error            { return e.profiles.Remove(id) }
func (e *Engine) Settings() (model.Settings, error)        { return e.settings.Get() }
func (e *Engine) SetSettings(v model.Settings) (model.Settings, error) {
	saved, err := e.settings.Set(v)
	if err == nil {
		e.transfers.SettingsChanged()
	}
	return saved, err
}

// SaveProfile is typed so profile secrets never pass through a generic
// serializer and do not create avoidable password/passphrase copies.
func (e *Engine) SaveProfile(in model.ProfileInput) (saved model.PublicProfile, callErr error) {
	defer recoverTypedCall(&callErr)
	return e.profiles.Save(in)
}

// Connect is typed for the same reason as SaveProfile: connection secrets stay
// inside the process and are not serialized into an intermediate payload.
func (e *Engine) Connect(ctx context.Context, profileID string, cfg model.ConnectionConfig, trustFingerprint string, rememberFingerprint bool) (result remote.ConnectResult, callErr error) {
	defer recoverTypedCall(&callErr)
	result, callErr = e.remote.Connect(ctx, profileID, cfg, trustFingerprint, rememberFingerprint)
	if callErr == nil && result.Connected {
		e.transfers.Enable()
	}
	return result, callErr
}

func (e *Engine) CancelPendingTrust() { e.remote.CancelPendingTrust() }

func (e *Engine) Disconnect(ctx context.Context) error {
	transferErr := e.transfers.DisableAndCancel(ctx, "Otkazano prekidom veze")
	remoteErr := e.remote.Disconnect(ctx)
	return errors.Join(transferErr, remoteErr)
}

func (e *Engine) Probe(ctx context.Context) error { return e.remote.Probe(ctx) }

func (e *Engine) LocalList(ctx context.Context, path string) (string, []model.Item, error) {
	return e.local.ListContext(ctx, path)
}
func (e *Engine) LocalMkdir(base, name string) error { return e.local.Mkdir(base, name) }
func (e *Engine) LocalRename(base, oldName, newName string) error {
	return e.local.Rename(base, oldName, newName)
}
func (e *Engine) LocalDelete(base, name string) error { return e.local.Delete(base, name) }

func (e *Engine) RemoteList(ctx context.Context, path string) ([]model.Item, error) {
	s, opCtx, release, err := e.remote.Operation(ctx)
	if err != nil {
		return nil, err
	}
	defer release()
	return s.List(opCtx, path)
}
func (e *Engine) RemoteMkdir(ctx context.Context, base, name string) error {
	s, opCtx, release, err := e.remote.Operation(ctx)
	if err != nil {
		return err
	}
	defer release()
	return s.Mkdir(opCtx, base, name)
}
func (e *Engine) RemoteRename(ctx context.Context, base, oldName, newName string) error {
	s, opCtx, release, err := e.remote.Operation(ctx)
	if err != nil {
		return err
	}
	defer release()
	return s.Rename(opCtx, base, oldName, newName)
}
func (e *Engine) RemoteDelete(ctx context.Context, base, name string, isDirectory bool) error {
	s, opCtx, release, err := e.remote.Operation(ctx)
	if err != nil {
		return err
	}
	defer release()
	return s.Delete(opCtx, base, name, isDirectory)
}
func (e *Engine) RemoteChmod(ctx context.Context, base, name, mode string) error {
	s, opCtx, release, err := e.remote.Operation(ctx)
	if err != nil {
		return err
	}
	defer release()
	return s.Chmod(opCtx, base, name, mode)
}

func (e *Engine) AddTransfer(direction, localPath, remotePath, localRoot string) (model.TransferJob, error) {
	if err := security.ValidateRemoteFilePath(remotePath); err != nil {
		return model.TransferJob{}, err
	}
	return e.transfers.AddBatchOne(transfer.Request{Direction: direction, LocalPath: localPath, RemotePath: remotePath, LocalRoot: localRoot})
}

type TransferRequest struct {
	Direction  string
	LocalPath  string
	RemotePath string
	LocalRoot  string
}

func (e *Engine) AddTransfers(requests []TransferRequest) ([]model.TransferJob, error) {
	batch := make([]transfer.Request, 0, len(requests))
	for _, r := range requests {
		if err := security.ValidateRemoteFilePath(r.RemotePath); err != nil {
			return nil, err
		}
		batch = append(batch, transfer.Request{Direction: r.Direction, LocalPath: r.LocalPath, RemotePath: r.RemotePath, LocalRoot: r.LocalRoot})
	}
	return e.transfers.AddBatch(batch)
}
func (e *Engine) AddTreeTransfer(ctx context.Context, direction, localPath, remotePath string) (TreeTransferResult, error) {
	return e.addTree(ctx, direction, localPath, remotePath)
}
func (e *Engine) PauseTransfers()                    { e.transfers.Pause() }
func (e *Engine) ResumeTransfers()                   { e.transfers.Resume() }
func (e *Engine) CancelTransfer(id string) error     { return e.transfers.Cancel(id) }
func (e *Engine) CancelTransfers(ids []string) error { return e.transfers.CancelBatch(ids) }
func (e *Engine) RetryTransfer(id string) error {
	return e.RetryTransfers([]string{id})
}
func (e *Engine) RetryTransfers(ids []string) error {
	return e.transfers.RetryBatch(ids)
}
func (e *Engine) ClearFinishedTransfers() { e.transfers.ClearFinished() }
func (e *Engine) TransferEvents(since int64) ([]transfer.Event, int64) {
	return e.transfers.Events(since)
}

func (e *Engine) Close() {
	e.closeOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
		defer cancel()
		_ = e.transfers.Shutdown(ctx)
		_ = e.remote.Disconnect(ctx)
	})
}

func DataDir() (string, error) {
	base, err := platform.LocalAppData()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, brand.Company, brand.ProductName), nil
}
