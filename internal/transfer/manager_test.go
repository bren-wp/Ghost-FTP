package transfer

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/bren-wp/Ghost-FTP/internal/config"
	"github.com/bren-wp/Ghost-FTP/internal/model"
	"github.com/bren-wp/Ghost-FTP/internal/remote"
)

type disconnectedProvider struct{}

func (disconnectedProvider) Operation(ctx context.Context) (remote.Session, context.Context, func(), error) {
	return nil, nil, func() {}, errors.New("nije uspostavljena veza")
}
func (disconnectedProvider) ConnectionIdentity() (string, error) {
	return "test-disconnected-endpoint", nil
}

func waitForStatus(t *testing.T, m *Manager, id, status string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		for _, j := range m.List() {
			if j.ID == id && j.Status == status {
				return
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("transfer %s did not reach status %s", id, status)
}

func TestRetryFailedTransfer(t *testing.T) {
	dir := t.TempDir()
	store := config.New(dir)
	settings := config.NewSettings(store)
	m := New(disconnectedProvider{}, settings)

	local := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(local, []byte("test"), 0600); err != nil {
		t.Fatal(err)
	}
	job, err := m.Add("upload", local, "/file.txt")
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	waitForStatus(t, m, job.ID, "failed")

	if err := m.Retry(job.ID); err != nil {
		t.Fatalf("Retry: %v", err)
	}
	// No remote session is attached, so a retried job must execute again and fail,
	// proving it was genuinely re-queued rather than left in the old terminal state.
	waitForStatus(t, m, job.ID, "failed")
}

func TestRetryMissingTransfer(t *testing.T) {
	dir := t.TempDir()
	store := config.New(dir)
	m := New(disconnectedProvider{}, config.NewSettings(store))
	if err := m.Retry("missing"); err == nil {
		t.Fatal("expected error for missing transfer")
	}
}

func TestAddBatchIsAtomicOnValidationFailure(t *testing.T) {
	dir := t.TempDir()
	store := config.New(dir)
	m := New(disconnectedProvider{}, config.NewSettings(store))
	m.Pause()

	_, err := m.AddBatch([]Request{
		{Direction: "upload", LocalPath: filepath.Join(dir, "a.txt"), RemotePath: "/a.txt"},
		{Direction: "upload", LocalPath: filepath.Join(dir, "b.txt"), RemotePath: "/../b.txt"},
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
	if got := len(m.List()); got != 0 {
		t.Fatalf("queue mutated after rejected batch: got %d jobs", got)
	}
}

func TestDisableAndCancelCancelsQueuedJobsAndClosesAdmission(t *testing.T) {
	dir := t.TempDir()
	store := config.New(dir)
	m := New(disconnectedProvider{}, config.NewSettings(store))
	m.Pause()

	jobs, err := m.AddBatch([]Request{
		{Direction: "upload", LocalPath: filepath.Join(dir, "a.txt"), RemotePath: "/a.txt"},
		{Direction: "download", LocalPath: filepath.Join(dir, "b.txt"), RemotePath: "/b.txt"},
	})
	if err != nil {
		t.Fatalf("AddBatch: %v", err)
	}
	if err := m.DisableAndCancel(context.Background(), "test disconnect"); err != nil {
		t.Fatalf("DisableAndCancel: %v", err)
	}
	if got := m.ActiveCount(); got != 0 {
		t.Fatalf("active count after disable = %d, want 0", got)
	}
	for _, got := range m.List() {
		for _, want := range jobs {
			if got.ID == want.ID && (got.Status != "cancelled" || got.Error != "test disconnect") {
				t.Fatalf("job %s = status %q error %q", got.ID, got.Status, got.Error)
			}
		}
	}
	if _, err := m.Add("upload", filepath.Join(dir, "c.txt"), "/c.txt"); err == nil {
		t.Fatal("expected queue admission to remain disabled after disconnect")
	}
	m.Enable()
	m.Pause()
	if _, err := m.Add("upload", filepath.Join(dir, "c.txt"), "/c.txt"); err != nil {
		t.Fatalf("queue did not reopen after Enable: %v", err)
	}
}

func TestReservationCannotCommitAcrossDisconnect(t *testing.T) {
	dir := t.TempDir()
	store := config.New(dir)
	m := New(disconnectedProvider{}, config.NewSettings(store))
	m.Pause()
	reservation, err := m.ReserveBatch([]Request{{Direction: "download", LocalPath: filepath.Join(dir, "a.txt"), RemotePath: "/a.txt"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := m.DisableAndCancel(context.Background(), "disconnect"); err != nil {
		t.Fatal(err)
	}
	if _, err := reservation.Commit(); err == nil {
		t.Fatal("expected reserved batch commit to fail after disconnect")
	}
	if got := len(m.List()); got != 0 {
		t.Fatalf("reservation leaked into queue: %d", got)
	}
}

func TestEventsReturnsSnapshotWhenCursorIsStale(t *testing.T) {
	dir := t.TempDir()
	store := config.New(dir)
	m := New(disconnectedProvider{}, config.NewSettings(store))
	m.Pause()

	for i := 0; i < 1100; i++ {
		_, err := m.Add("upload", filepath.Join(dir, "file.txt"), "/file.txt")
		if err != nil {
			t.Fatalf("Add #%d: %v", i, err)
		}
	}
	events, seq := m.Events(1)
	if seq == 0 || len(events) != 1 || events[0].Type != "state" {
		t.Fatalf("expected one state snapshot for stale cursor; seq=%d events=%#v", seq, events)
	}
	if len(events[0].Jobs) != 1100 {
		t.Fatalf("snapshot contains %d jobs, want 1100", len(events[0].Jobs))
	}
}

func TestBatchReservationDoesNotExposeJobsBeforeCommit(t *testing.T) {
	dir := t.TempDir()
	store := config.New(dir)
	m := New(disconnectedProvider{}, config.NewSettings(store))
	m.Pause()

	reservation, err := m.ReserveBatch([]Request{{Direction: "upload", LocalPath: filepath.Join(dir, "a.txt"), RemotePath: "/a.txt"}})
	if err != nil {
		t.Fatalf("ReserveBatch: %v", err)
	}
	if got := len(m.List()); got != 0 {
		t.Fatalf("reserved job became visible before commit: %d", got)
	}
	jobs, err := reservation.Commit()
	if err != nil {
		t.Fatalf("Commit: %v", err)
	}
	if len(jobs) != 1 || len(m.List()) != 1 {
		t.Fatalf("committed batch not visible: jobs=%d queue=%d", len(jobs), len(m.List()))
	}
	if _, err := reservation.Commit(); err == nil {
		t.Fatal("expected second commit to fail")
	}
}

func TestBatchReservationCancelReleasesCapacity(t *testing.T) {
	dir := t.TempDir()
	store := config.New(dir)
	m := New(disconnectedProvider{}, config.NewSettings(store))
	m.Pause()

	reservation, err := m.ReserveBatch([]Request{{Direction: "download", LocalPath: filepath.Join(dir, "a.txt"), RemotePath: "/a.txt"}})
	if err != nil {
		t.Fatalf("ReserveBatch: %v", err)
	}
	reservation.Cancel()
	if _, err := reservation.Commit(); err == nil {
		t.Fatal("expected cancelled reservation commit to fail")
	}
	if _, err := m.AddBatch([]Request{{Direction: "download", LocalPath: filepath.Join(dir, "b.txt"), RemotePath: "/b.txt"}}); err != nil {
		t.Fatalf("capacity was not released after cancel: %v", err)
	}
}

func TestFailedJobDoesNotExposeInternalToolDetails(t *testing.T) {
	dir := t.TempDir()
	store := config.New(dir)
	m := New(disconnectedProvider{}, config.NewSettings(store))
	job, err := m.Add("upload", filepath.Join(dir, "missing.txt"), "/missing.txt")
	if err != nil {
		t.Fatal(err)
	}
	waitForStatus(t, m, job.ID, "failed")
	for _, got := range m.List() {
		if got.ID == job.ID {
			if got.Error == "" {
				t.Fatal("expected safe user-facing error")
			}
			if len(got.Error) > 300 {
				t.Fatalf("error too long: %d", len(got.Error))
			}
			return
		}
	}
	t.Fatal("job missing")
}

func TestReservationFromOldConnectionGenerationCannotCommitAfterReconnect(t *testing.T) {
	dir := t.TempDir()
	store := config.New(dir)
	m := New(disconnectedProvider{}, config.NewSettings(store))
	m.Enable()
	reservation, err := m.ReserveBatch([]Request{{Direction: "download", LocalPath: filepath.Join(dir, "old.txt"), RemotePath: "/old.txt"}})
	if err != nil {
		t.Fatalf("ReserveBatch: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := m.DisableAndCancel(ctx, "disconnect"); err != nil {
		t.Fatalf("DisableAndCancel: %v", err)
	}
	m.Enable()
	if _, err := reservation.Commit(); err == nil {
		t.Fatal("reservation created for an old connection generation must not commit after reconnect")
	}
	if got := len(m.List()); got != 0 {
		t.Fatalf("stale reservation mutated queue: %d jobs", got)
	}
}

func TestEnableEmitsSingleStateEvent(t *testing.T) {
	dir := t.TempDir()
	store := config.New(dir)
	m := New(disconnectedProvider{}, config.NewSettings(store))
	_, before := m.Events(0)
	m.Enable()
	events, _ := m.Events(before)
	states := 0
	for _, e := range events {
		if e.Type == "state" {
			states++
		}
	}
	if states != 1 {
		t.Fatalf("Enable should emit exactly one state event, got %d (%#v)", states, events)
	}
}

type panicSession struct{}

func (panicSession) Protocol() string                                     { return "ftp" }
func (panicSession) Host() string                                         { return "example.test" }
func (panicSession) Port() int                                            { return 21 }
func (panicSession) List(context.Context, string) ([]model.Item, error)   { return nil, nil }
func (panicSession) Mkdir(context.Context, string, string) error          { return nil }
func (panicSession) Rename(context.Context, string, string, string) error { return nil }
func (panicSession) Delete(context.Context, string, string, bool) error   { return nil }
func (panicSession) Chmod(context.Context, string, string, string) error  { return nil }
func (panicSession) Upload(context.Context, string, string, remote.TransferOptions) error {
	panic("simulated transfer panic")
}
func (panicSession) Download(context.Context, string, string, remote.TransferOptions) error {
	return nil
}
func (panicSession) Close() error { return nil }

type staticProvider struct {
	session  remote.Session
	identity string
}

func (p staticProvider) Operation(ctx context.Context) (remote.Session, context.Context, func(), error) {
	return p.session, ctx, func() {}, nil
}
func (p staticProvider) ConnectionIdentity() (string, error) {
	if p.identity != "" {
		return p.identity, nil
	}
	return "test-connection", nil
}

func TestWorkerPanicFailsOnlyJobAndReleasesSlot(t *testing.T) {
	dir := t.TempDir()
	store := config.New(dir)
	settings := config.NewSettings(store)
	m := New(staticProvider{session: panicSession{}}, settings)
	job, err := m.Add("upload", filepath.Join(dir, "file.txt"), "/file.txt")
	if err != nil {
		t.Fatal(err)
	}
	waitForStatus(t, m, job.ID, "failed")
	if got := m.ActiveCount(); got != 0 {
		t.Fatalf("active jobs after recovered panic = %d, want 0", got)
	}
	for _, got := range m.List() {
		if got.ID == job.ID && got.Error == "" {
			t.Fatal("recovered panic should become a safe job error")
		}
	}
	// The worker slot must be reusable after recovery.
	second, err := m.Add("download", filepath.Join(dir, "download.txt"), "/download.txt")
	if err != nil {
		t.Fatal(err)
	}
	waitForStatus(t, m, second.ID, "done")
}

type flakySession struct {
	mu       sync.Mutex
	attempts int
}

func (s *flakySession) Protocol() string                                     { return "ftp" }
func (s *flakySession) Host() string                                         { return "example.test" }
func (s *flakySession) Port() int                                            { return 21 }
func (s *flakySession) List(context.Context, string) ([]model.Item, error)   { return nil, nil }
func (s *flakySession) Mkdir(context.Context, string, string) error          { return nil }
func (s *flakySession) Rename(context.Context, string, string, string) error { return nil }
func (s *flakySession) Delete(context.Context, string, string, bool) error   { return nil }
func (s *flakySession) Chmod(context.Context, string, string, string) error  { return nil }
func (s *flakySession) Upload(context.Context, string, string, remote.TransferOptions) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.attempts++
	if s.attempts == 1 {
		return errors.New("connection reset by peer")
	}
	return nil
}
func (s *flakySession) Download(context.Context, string, string, remote.TransferOptions) error {
	return nil
}
func (s *flakySession) Close() error { return nil }

func TestAutomaticRetryCanRecoverTransientTransferFailure(t *testing.T) {
	dir := t.TempDir()
	store := config.New(dir)
	settings := config.NewSettings(store)
	if _, err := settings.Set(model.Settings{
		Parallelism:           2,
		BackupBeforeOverwrite: true,
		ConfirmDelete:         true,
		AutoRetryCount:        1,
		RetryDelaySeconds:     1,
	}); err != nil {
		t.Fatal(err)
	}
	session := &flakySession{}
	m := New(staticProvider{session: session}, settings)
	job, err := m.Add("upload", filepath.Join(dir, "file.txt"), "/file.txt")
	if err != nil {
		t.Fatal(err)
	}
	waitForStatus(t, m, job.ID, "done")
	for _, got := range m.List() {
		if got.ID == job.ID {
			if got.Attempts != 2 {
				t.Fatalf("attempts=%d want 2", got.Attempts)
			}
			return
		}
	}
	t.Fatal("job missing")
}

func TestCancelBatchValidatesWholeSelectionBeforeMutation(t *testing.T) {
	dir := t.TempDir()
	store := config.New(dir)
	m := New(disconnectedProvider{}, config.NewSettings(store))
	m.Pause()
	jobs, err := m.AddBatch([]Request{
		{Direction: "upload", LocalPath: filepath.Join(dir, "a.txt"), RemotePath: "/a.txt"},
		{Direction: "upload", LocalPath: filepath.Join(dir, "b.txt"), RemotePath: "/b.txt"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := m.CancelBatch([]string{jobs[0].ID, "missing"}); err == nil {
		t.Fatal("expected invalid batch selection to fail")
	}
	for _, got := range m.List() {
		if got.Status != "queued" {
			t.Fatalf("queue mutated after rejected cancel batch: %#v", m.List())
		}
	}
	if err := m.CancelBatch([]string{jobs[0].ID, jobs[1].ID}); err != nil {
		t.Fatal(err)
	}
	for _, got := range m.List() {
		if got.Status != "cancelled" {
			t.Fatalf("job %s status=%s want cancelled", got.ID, got.Status)
		}
	}
}

func TestRetryBatchValidatesWholeSelectionBeforeMutation(t *testing.T) {
	dir := t.TempDir()
	store := config.New(dir)
	m := New(disconnectedProvider{}, config.NewSettings(store))
	m.Pause()
	jobs, err := m.AddBatch([]Request{
		{Direction: "upload", LocalPath: filepath.Join(dir, "a.txt"), RemotePath: "/a.txt"},
		{Direction: "upload", LocalPath: filepath.Join(dir, "b.txt"), RemotePath: "/b.txt"},
	})
	if err != nil {
		t.Fatal(err)
	}
	m.mu.Lock()
	m.jobs[0].Status = "failed"
	m.jobs[0].Error = "failed"
	m.jobs[1].Status = "done"
	m.mu.Unlock()
	if err := m.RetryBatch([]string{jobs[0].ID, jobs[1].ID}); err == nil {
		t.Fatal("expected mixed terminal states to reject retry batch")
	}
	got := m.List()
	if got[0].Status != "failed" || got[1].Status != "done" {
		t.Fatalf("queue mutated after rejected retry batch: %#v", got)
	}
	m.mu.Lock()
	m.jobs[1].Status = "cancelled"
	m.mu.Unlock()
	if err := m.RetryBatch([]string{jobs[0].ID, jobs[1].ID}); err != nil {
		t.Fatal(err)
	}
	for _, job := range m.List() {
		if job.Status != "queued" {
			t.Fatalf("job %s status=%s want queued", job.ID, job.Status)
		}
	}
}

func TestRetryCannotCrossConnectionIdentity(t *testing.T) {
	dir := t.TempDir()
	settings := config.NewSettings(config.New(dir))
	provider := &staticProvider{session: panicSession{}, identity: "server-a"}
	m := New(provider, settings)
	m.Pause()
	jobs, err := m.AddBatch([]Request{{Direction: "upload", LocalPath: filepath.Join(dir, "private.txt"), RemotePath: "/private.txt"}})
	if err != nil {
		t.Fatal(err)
	}
	m.mu.Lock()
	m.jobs[0].Status = "failed"
	m.jobs[0].Error = "failed"
	m.mu.Unlock()
	provider.identity = "server-b"
	if err := m.RetryBatch([]string{jobs[0].ID}); err == nil {
		t.Fatal("transfer from server-a must not be retryable on server-b")
	}
	got := m.List()
	if len(got) != 1 || got[0].Status != "failed" {
		t.Fatalf("rejected cross-server retry mutated queue: %#v", got)
	}
}

type resultSession struct {
	mu          sync.Mutex
	uploadCalls int
	uploadErr   error
	lastOptions remote.TransferOptions
}

func (s *resultSession) Protocol() string                                     { return "ftp" }
func (s *resultSession) Host() string                                         { return "example.test" }
func (s *resultSession) Port() int                                            { return 21 }
func (s *resultSession) List(context.Context, string) ([]model.Item, error)   { return nil, nil }
func (s *resultSession) Mkdir(context.Context, string, string) error          { return nil }
func (s *resultSession) Rename(context.Context, string, string, string) error { return nil }
func (s *resultSession) Delete(context.Context, string, string, bool) error   { return nil }
func (s *resultSession) Chmod(context.Context, string, string, string) error  { return nil }
func (s *resultSession) Upload(_ context.Context, _, _ string, options remote.TransferOptions) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.uploadCalls++
	s.lastOptions = options
	return s.uploadErr
}
func (s *resultSession) Download(context.Context, string, string, remote.TransferOptions) error {
	return nil
}
func (s *resultSession) Close() error { return nil }

func TestSkipExistingFinishesAsSkipped(t *testing.T) {
	dir := t.TempDir()
	settings := config.NewSettings(config.New(dir))
	if _, err := settings.Set(model.Settings{Parallelism: 1, BackupBeforeOverwrite: true, ConfirmDelete: true, AutoRetryCount: 3, RetryDelaySeconds: 1, SkipExisting: true}); err != nil {
		t.Fatal(err)
	}
	session := &resultSession{uploadErr: remote.ErrSkipped}
	m := New(staticProvider{session: session}, settings)
	job, err := m.Add("upload", filepath.Join(dir, "file.txt"), "/file.txt")
	if err != nil {
		t.Fatal(err)
	}
	waitForStatus(t, m, job.ID, "skipped")
	session.mu.Lock()
	calls, options := session.uploadCalls, session.lastOptions
	session.mu.Unlock()
	if calls != 1 {
		t.Fatalf("skipped transfer calls=%d want 1", calls)
	}
	if !options.SkipExisting || !options.KeepBackup {
		t.Fatalf("transfer options not propagated: %+v", options)
	}
	m.ClearFinished()
	if got := m.List(); len(got) != 0 {
		t.Fatalf("skipped job should be clearable: %#v", got)
	}
}

func TestPermanentTransferFailureIsNotAutomaticallyRetried(t *testing.T) {
	dir := t.TempDir()
	settings := config.NewSettings(config.New(dir))
	if _, err := settings.Set(model.Settings{Parallelism: 1, BackupBeforeOverwrite: true, ConfirmDelete: true, AutoRetryCount: 3, RetryDelaySeconds: 1}); err != nil {
		t.Fatal(err)
	}
	session := &resultSession{uploadErr: errors.New("permission denied")}
	m := New(staticProvider{session: session}, settings)
	job, err := m.Add("upload", filepath.Join(dir, "file.txt"), "/file.txt")
	if err != nil {
		t.Fatal(err)
	}
	waitForStatus(t, m, job.ID, "failed")
	session.mu.Lock()
	calls := session.uploadCalls
	session.mu.Unlock()
	if calls != 1 {
		t.Fatalf("permanent error retried %d times, want exactly one attempt", calls)
	}
}

func TestLocalRootIsRevalidatedWhenQueuedJobStarts(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	link := filepath.Join(root, "linked")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlink not available: %v", err)
	}
	settings := config.NewSettings(config.New(t.TempDir()))
	session := &resultSession{}
	m := New(staticProvider{session: session}, settings)
	m.Pause()
	jobs, err := m.AddBatch([]Request{{Direction: "download", LocalPath: filepath.Join(link, "remote.txt"), RemotePath: "/remote.txt", LocalRoot: root}})
	if err != nil {
		t.Fatal(err)
	}
	m.Resume()
	waitForStatus(t, m, jobs[0].ID, "failed")
	session.mu.Lock()
	calls := session.uploadCalls
	session.mu.Unlock()
	if calls != 0 {
		t.Fatalf("remote adapter should not run after local-root validation failure; uploadCalls=%d", calls)
	}
}

type blockingSession struct {
	started chan string
	release chan struct{}
}

func (s *blockingSession) Protocol() string                                     { return "ftp" }
func (s *blockingSession) Host() string                                         { return "example.test" }
func (s *blockingSession) Port() int                                            { return 21 }
func (s *blockingSession) List(context.Context, string) ([]model.Item, error)   { return nil, nil }
func (s *blockingSession) Mkdir(context.Context, string, string) error          { return nil }
func (s *blockingSession) Rename(context.Context, string, string, string) error { return nil }
func (s *blockingSession) Delete(context.Context, string, string, bool) error   { return nil }
func (s *blockingSession) Chmod(context.Context, string, string, string) error  { return nil }
func (s *blockingSession) Upload(ctx context.Context, local, _ string, _ remote.TransferOptions) error {
	select {
	case s.started <- local:
	case <-ctx.Done():
		return ctx.Err()
	}
	select {
	case <-s.release:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
func (s *blockingSession) Download(ctx context.Context, remotePath, _ string, o remote.TransferOptions) error {
	return s.Upload(ctx, remotePath, "", o)
}
func (s *blockingSession) Close() error { return nil }

func TestSettingsChangedAppliesHigherParallelismImmediately(t *testing.T) {
	dir := t.TempDir()
	settings := config.NewSettings(config.New(dir))
	base := model.Settings{Parallelism: 1, BackupBeforeOverwrite: true, ConfirmDelete: true, RetryDelaySeconds: 1}
	if _, err := settings.Set(base); err != nil {
		t.Fatal(err)
	}
	session := &blockingSession{started: make(chan string, 4), release: make(chan struct{})}
	m := New(staticProvider{session: session}, settings)
	for i := 0; i < 3; i++ {
		if _, err := m.Add("upload", filepath.Join(dir, string(rune('a'+i))+".txt"), "/file.txt"); err != nil {
			t.Fatal(err)
		}
	}
	select {
	case <-session.started:
	case <-time.After(time.Second):
		t.Fatal("first worker did not start")
	}
	select {
	case got := <-session.started:
		t.Fatalf("parallelism=1 unexpectedly started second worker: %s", got)
	case <-time.After(50 * time.Millisecond):
	}
	base.Parallelism = 3
	if _, err := settings.Set(base); err != nil {
		t.Fatal(err)
	}
	m.SettingsChanged()
	for i := 0; i < 2; i++ {
		select {
		case <-session.started:
		case <-time.After(time.Second):
			t.Fatal("increased parallelism did not start waiting worker immediately")
		}
	}
	close(session.release)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := m.Shutdown(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestClearFinishedPurgesFinishedJobMetadataFromEventBuffer(t *testing.T) {
	dir := t.TempDir()
	settings := config.NewSettings(config.New(dir))
	if _, err := settings.Set(model.Settings{Parallelism: 1, BackupBeforeOverwrite: true, ConfirmDelete: true}); err != nil {
		t.Fatal(err)
	}
	sensitiveLocal := filepath.Join(dir, "private-customer-document.txt")
	sensitiveRemote := "/private/customer/document.txt"
	m := New(staticProvider{session: &resultSession{}}, settings)
	job, err := m.Add("upload", sensitiveLocal, sensitiveRemote)
	if err != nil {
		t.Fatal(err)
	}
	waitForStatus(t, m, job.ID, "done")
	m.ClearFinished()

	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.jobs) != 0 {
		t.Fatalf("finished jobs were not removed: %#v", m.jobs)
	}
	for _, event := range m.events {
		if event.Job != nil && (event.Job.LocalPath == sensitiveLocal || event.Job.RemotePath == sensitiveRemote) {
			t.Fatalf("finished path metadata survived ClearFinished in job event: %#v", event)
		}
		for _, eventJob := range event.Jobs {
			if eventJob.LocalPath == sensitiveLocal || eventJob.RemotePath == sensitiveRemote {
				t.Fatalf("finished path metadata survived ClearFinished in state event: %#v", event)
			}
		}
	}
}
