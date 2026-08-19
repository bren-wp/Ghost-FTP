package transfer

import (
	"brendigo.com/byftp/internal/config"
	"brendigo.com/byftp/internal/model"
	"brendigo.com/byftp/internal/remote"
	"brendigo.com/byftp/internal/security"
	"brendigo.com/byftp/internal/usererror"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Event struct {
	Seq    int64               `json:"seq"`
	Type   string              `json:"type"`
	Job    *model.TransferJob  `json:"job,omitempty"`
	Jobs   []model.TransferJob `json:"jobs,omitempty"`
	Paused bool                `json:"paused,omitempty"`
}

const maxTransferJobs = 20000

type operationProvider interface {
	Operation(context.Context) (remote.Session, context.Context, func(), error)
	ConnectionIdentity() (string, error)
}

type Manager struct {
	mu             sync.RWMutex
	wg             sync.WaitGroup
	jobs           []model.TransferJob
	paused         bool
	accepting      bool
	closed         bool
	running        int
	reserved       int
	generation     uint64
	cancels        map[string]context.CancelFunc
	jobConnections map[string]string
	events         []Event
	seq            int64
	remote         operationProvider
	settings       *config.SettingsStore
}

func New(r operationProvider, s *config.SettingsStore) *Manager {
	return &Manager{remote: r, settings: s, accepting: true, cancels: map[string]context.CancelFunc{}, jobConnections: map[string]string{}}
}

func id() (string, error) {
	b := make([]byte, 10)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func cloneEvent(e Event) Event {
	out := e
	if e.Job != nil {
		job := *e.Job
		out.Job = &job
	}
	if e.Jobs != nil {
		out.Jobs = append([]model.TransferJob(nil), e.Jobs...)
	}
	return out
}

func selectedIDs(ids []string) (map[string]struct{}, error) {
	if len(ids) == 0 {
		return nil, errors.New("nije odabran nijedan prijenos")
	}
	wanted := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			return nil, errors.New("neispravan prijenos u odabiru")
		}
		wanted[id] = struct{}{}
	}
	return wanted, nil
}

func (m *Manager) emitLocked(e Event) {
	m.seq++
	e.Seq = m.seq
	m.events = append(m.events, cloneEvent(e))
	if len(m.events) > 1000 {
		m.events = append([]Event(nil), m.events[len(m.events)-500:]...)
	}
}

type Request struct {
	Direction  string
	LocalPath  string
	RemotePath string
	LocalRoot  string
}

func validateRequest(r Request) error {
	if r.Direction != "upload" && r.Direction != "download" {
		return errors.New("neispravan smjer prijenosa")
	}
	if strings.TrimSpace(r.LocalPath) == "" || strings.TrimSpace(r.RemotePath) == "" {
		return errors.New("putanje prijenosa su obavezne")
	}
	if len(r.LocalPath) > 32767 || strings.ContainsAny(r.LocalPath, "\x00\r\n") {
		return errors.New("neispravna lokalna putanja")
	}
	if _, err := filepath.Abs(r.LocalPath); err != nil {
		return errors.New("neispravna lokalna putanja")
	}
	if r.LocalRoot != "" {
		if len(r.LocalRoot) > 32767 || strings.ContainsAny(r.LocalRoot, "\x00\r\n") {
			return errors.New("neispravan lokalni korijen prijenosa")
		}
		if _, err := filepath.Abs(r.LocalRoot); err != nil {
			return errors.New("neispravan lokalni korijen prijenosa")
		}
	}
	if err := security.ValidateRemotePath(r.RemotePath); err != nil {
		return err
	}
	return nil
}

func (m *Manager) Add(direction, localPath, remotePath string) (model.TransferJob, error) {
	return m.AddBatchOne(Request{Direction: direction, LocalPath: localPath, RemotePath: remotePath})
}

func (m *Manager) AddBatchOne(request Request) (model.TransferJob, error) {
	jobs, err := m.AddBatch([]Request{request})
	if err != nil {
		return model.TransferJob{}, err
	}
	return jobs[0], nil
}

// BatchReservation reserves queue capacity without starting any work. It lets
// higher-level folder transfers validate the complete operation and prepare
// required directories before jobs become visible/runnable.
type BatchReservation struct {
	m            *Manager
	jobs         []model.TransferJob
	generation   uint64
	connectionID string
	done         bool
}

func buildJobs(requests []Request) ([]model.TransferJob, error) {
	if len(requests) == 0 {
		return nil, errors.New("nema prijenosa za dodavanje")
	}
	jobs := make([]model.TransferJob, 0, len(requests))
	now := time.Now().UTC().Format(time.RFC3339)
	for i, r := range requests {
		if err := validateRequest(r); err != nil {
			return nil, errors.New("prijenos #" + strconv.Itoa(i+1) + ": " + err.Error())
		}
		jobID, err := id()
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, model.TransferJob{
			ID: jobID, Direction: r.Direction, LocalPath: r.LocalPath, RemotePath: r.RemotePath, LocalRoot: r.LocalRoot,
			Status: "queued", CreatedAt: now,
		})
	}
	return jobs, nil
}

func (m *Manager) ReserveBatch(requests []Request) (*BatchReservation, error) {
	jobs, err := buildJobs(requests)
	if err != nil {
		return nil, err
	}

	// Capture the queue generation before asking the remote manager for its
	// connection identity. The identity lookup deliberately runs without m.mu to
	// avoid lock-order coupling with the remote manager. After the lookup we must
	// prove the generation is still identical before reserving any capacity.
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return nil, errors.New("transfer manager je zatvoren")
	}
	if !m.accepting {
		m.mu.Unlock()
		return nil, errors.New("red prijenosa trenutačno ne prima nove poslove")
	}
	generation := m.generation
	m.mu.Unlock()

	connectionID, err := m.remote.ConnectionIdentity()
	if err != nil {
		return nil, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if m.generation != generation {
		return nil, errors.New("veza se promijenila tijekom pripreme prijenosa")
	}
	if m.closed {
		return nil, errors.New("transfer manager je zatvoren")
	}
	if !m.accepting {
		return nil, errors.New("red prijenosa trenutačno ne prima nove poslove")
	}
	if len(m.jobs)+m.reserved+len(jobs) > maxTransferJobs {
		return nil, errors.New("red prijenosa je prevelik; očistite završene prijenose prije dodavanja novih")
	}
	m.reserved += len(jobs)
	return &BatchReservation{m: m, jobs: jobs, generation: generation, connectionID: connectionID}, nil
}

func (b *BatchReservation) Cancel() {
	if b == nil || b.m == nil {
		return
	}
	m := b.m
	m.mu.Lock()
	if !b.done {
		m.reserved -= len(b.jobs)
		if m.reserved < 0 {
			m.reserved = 0
		}
		b.done = true
	}
	m.mu.Unlock()
}

func (b *BatchReservation) Commit() ([]model.TransferJob, error) {
	if b == nil || b.m == nil {
		return nil, errors.New("neispravna rezervacija prijenosa")
	}
	m := b.m
	m.mu.Lock()
	if b.done {
		m.mu.Unlock()
		return nil, errors.New("rezervacija prijenosa više nije aktivna")
	}
	m.reserved -= len(b.jobs)
	if m.reserved < 0 {
		m.reserved = 0
	}
	b.done = true
	if b.generation != m.generation {
		m.mu.Unlock()
		return nil, errors.New("veza se promijenila prije dodavanja prijenosa")
	}
	if m.closed {
		m.mu.Unlock()
		return nil, errors.New("transfer manager je zatvoren")
	}
	if !m.accepting {
		m.mu.Unlock()
		return nil, errors.New("veza je prekinuta prije dodavanja prijenosa")
	}
	m.jobs = append(m.jobs, b.jobs...)
	for i := range b.jobs {
		j := b.jobs[i]
		m.jobConnections[j.ID] = b.connectionID
		m.emitLocked(Event{Type: "job", Job: &j})
	}
	out := append([]model.TransferJob(nil), b.jobs...)
	m.mu.Unlock()
	m.pump()
	return out, nil
}

// AddBatch validates the complete set before mutating the queue, preventing
// partially queued folder transfers when one later item is invalid.
func (m *Manager) AddBatch(requests []Request) ([]model.TransferJob, error) {
	reservation, err := m.ReserveBatch(requests)
	if err != nil {
		return nil, err
	}
	return reservation.Commit()
}

func (m *Manager) List() []model.TransferJob {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]model.TransferJob(nil), m.jobs...)
}

func (m *Manager) Events(since int64) ([]Event, int64) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if since > 0 && len(m.events) > 0 && since < m.events[0].Seq-1 {
		jobs := append([]model.TransferJob(nil), m.jobs...)
		return []Event{{Seq: m.seq, Type: "state", Jobs: jobs, Paused: m.paused}}, m.seq
	}
	out := make([]Event, 0, len(m.events))
	for _, e := range m.events {
		if e.Seq > since {
			out = append(out, cloneEvent(e))
		}
	}
	return out, m.seq
}

// SettingsChanged reapplies queue scheduling immediately after runtime settings
// change. In particular, increasing parallelism should start waiting jobs without
// requiring an unrelated transfer to finish first.
func (m *Manager) SettingsChanged() {
	m.pump()
}

func (m *Manager) Pause() {
	m.mu.Lock()
	m.paused = true
	jobs := append([]model.TransferJob(nil), m.jobs...)
	m.emitLocked(Event{Type: "state", Jobs: jobs, Paused: true})
	m.mu.Unlock()
}

func (m *Manager) Resume() {
	m.mu.Lock()
	if m.closed || !m.accepting {
		m.mu.Unlock()
		return
	}
	m.paused = false
	jobs := append([]model.TransferJob(nil), m.jobs...)
	m.emitLocked(Event{Type: "state", Jobs: jobs, Paused: false})
	m.mu.Unlock()
	m.pump()
}

func (m *Manager) Cancel(id string) error {
	return m.CancelBatch([]string{id})
}

func (m *Manager) CancelBatch(ids []string) error {
	wanted, err := selectedIDs(ids)
	if err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	indices := make([]int, 0, len(wanted))
	for i := range m.jobs {
		if _, ok := wanted[m.jobs[i].ID]; !ok {
			continue
		}
		if m.jobs[i].Status != "queued" && m.jobs[i].Status != "running" {
			return errors.New("jedan od odabranih prijenosa više nije aktivan")
		}
		indices = append(indices, i)
	}
	if len(indices) != len(wanted) {
		return errors.New("jedan od odabranih prijenosa nije pronađen")
	}
	for _, i := range indices {
		job := &m.jobs[i]
		if job.Status == "queued" {
			job.Status = "cancelled"
			job.Error = "Otkazano"
			j := *job
			m.emitLocked(Event{Type: "job", Job: &j})
			continue
		}
		if c := m.cancels[job.ID]; c != nil {
			c()
		}
	}
	return nil
}

func (m *Manager) ActiveCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	n := 0
	for _, job := range m.jobs {
		if job.Status == "queued" || job.Status == "running" {
			n++
		}
	}
	return n
}

func (m *Manager) Retry(id string) error {
	return m.RetryBatch([]string{id})
}

func (m *Manager) RetryBatch(ids []string) error {
	wanted, err := selectedIDs(ids)
	if err != nil {
		return err
	}

	// Retry must bind the remote identity to the same queue generation. A
	// disconnect/reconnect can otherwise return the old connection ID and then
	// let us mutate jobs after generation already advanced to the new session.
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return errors.New("transfer manager je zatvoren")
	}
	if !m.accepting {
		m.mu.Unlock()
		return errors.New("povežite se s poslužiteljem prije ponavljanja prijenosa")
	}
	generation := m.generation
	m.mu.Unlock()

	currentConnectionID, err := m.remote.ConnectionIdentity()
	if err != nil {
		return err
	}

	m.mu.Lock()
	if m.generation != generation {
		m.mu.Unlock()
		return errors.New("veza se promijenila tijekom pripreme ponavljanja prijenosa")
	}
	if m.closed {
		m.mu.Unlock()
		return errors.New("transfer manager je zatvoren")
	}
	if !m.accepting {
		m.mu.Unlock()
		return errors.New("povežite se s poslužiteljem prije ponavljanja prijenosa")
	}
	indices := make([]int, 0, len(wanted))
	for i := range m.jobs {
		if _, ok := wanted[m.jobs[i].ID]; !ok {
			continue
		}
		if m.jobs[i].Status != "failed" && m.jobs[i].Status != "cancelled" {
			m.mu.Unlock()
			return errors.New("jedan od odabranih prijenosa nije moguće ponoviti")
		}
		if m.jobConnections[m.jobs[i].ID] != currentConnectionID {
			m.mu.Unlock()
			return errors.New("odabrani prijenos pripada drugoj vezi; ponovno odaberite datoteku za ovaj poslužitelj")
		}
		indices = append(indices, i)
	}
	if len(indices) != len(wanted) {
		m.mu.Unlock()
		return errors.New("jedan od odabranih prijenosa nije pronađen")
	}
	for _, i := range indices {
		m.jobs[i].Status = "queued"
		m.jobs[i].Error = ""
		m.jobs[i].Progress = 0
		m.jobs[i].Attempts = 0
		j := m.jobs[i]
		m.emitLocked(Event{Type: "job", Job: &j})
	}
	m.mu.Unlock()
	m.pump()
	return nil
}

func (m *Manager) ClearFinished() {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]model.TransferJob, 0, len(m.jobs))
	for _, j := range m.jobs {
		if j.Status != "done" && j.Status != "failed" && j.Status != "cancelled" && j.Status != "skipped" {
			out = append(out, j)
			continue
		}
		delete(m.jobConnections, j.ID)
	}
	for i := range m.events {
		m.events[i] = Event{}
	}
	m.events = nil
	m.jobs = out
	m.emitLocked(Event{Type: "state", Jobs: append([]model.TransferJob(nil), m.jobs...), Paused: m.paused})
}

func (m *Manager) pump() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.paused || m.closed || !m.accepting {
		return
	}
	settings, _ := m.settings.Get()
	limit := settings.Parallelism
	if limit < 1 {
		limit = 2
	}
	for m.running < limit {
		idx := -1
		for i := range m.jobs {
			if m.jobs[i].Status == "queued" {
				idx = i
				break
			}
		}
		if idx < 0 {
			return
		}
		m.jobs[idx].Status = "running"
		m.jobs[idx].Progress = 0
		j := m.jobs[idx]
		m.running++
		ctx, cancel := context.WithCancel(context.Background())
		m.cancels[j.ID] = cancel
		m.emitLocked(Event{Type: "job", Job: &j})
		m.wg.Add(1)
		go func(jobID string) {
			defer m.wg.Done()
			m.execute(ctx, jobID)
		}(j.ID)
	}
}

func (m *Manager) jobSnapshot(id string) (model.TransferJob, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, job := range m.jobs {
		if job.ID == id {
			return job, true
		}
	}
	return model.TransferJob{}, false
}

func (m *Manager) finishJob(ctx context.Context, id string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.jobs {
		if m.jobs[i].ID != id {
			continue
		}
		// The transfer result is authoritative. A disconnect/cancel can race with
		// worker completion after the remote adapter has already returned success.
		// In that narrow window the completed file must stay marked as done instead
		// of being rewritten to cancelled merely because ctx was cancelled later.
		if errors.Is(err, remote.ErrSkipped) {
			m.jobs[i].Status = "skipped"
			m.jobs[i].Progress = 100
			m.jobs[i].Error = "Datoteka već postoji"
		} else if err == nil {
			m.jobs[i].Status = "done"
			m.jobs[i].Progress = 100
		} else if ctx.Err() != nil || errors.Is(err, context.Canceled) {
			m.jobs[i].Status = "cancelled"
			m.jobs[i].Error = "Otkazano"
		} else {
			m.jobs[i].Status = "failed"
			m.jobs[i].Error = usererror.Message(err, "Prijenos nije uspio. Provjerite vezu i pokušajte ponovno.")
		}
		j := m.jobs[i]
		m.emitLocked(Event{Type: "job", Job: &j})
		break
	}
	delete(m.cancels, id)
	if m.running > 0 {
		m.running--
	}
}

func (m *Manager) setAttempt(id string, attempt int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.jobs {
		if m.jobs[i].ID != id {
			continue
		}
		m.jobs[i].Attempts = attempt
		j := m.jobs[i]
		m.emitLocked(Event{Type: "job", Job: &j})
		return
	}
}

func waitRetryDelay(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (m *Manager) runAttempt(ctx context.Context, job model.TransferJob, settings model.Settings) error {
	if job.LocalRoot != "" {
		if err := security.EnsureLocalWithinRoot(job.LocalRoot, job.LocalPath); err != nil {
			return err
		}
	}
	sess, opCtx, release, err := m.remote.Operation(ctx)
	if err != nil {
		return err
	}
	defer release()
	options := remote.TransferOptions{KeepBackup: settings.BackupBeforeOverwrite, SkipExisting: settings.SkipExisting}
	if job.Direction == "upload" {
		return sess.Upload(opCtx, job.LocalPath, job.RemotePath, options)
	}
	return sess.Download(opCtx, job.RemotePath, job.LocalPath, options)
}

func (m *Manager) execute(ctx context.Context, id string) {
	var runErr error
	defer func() {
		if recover() != nil {
			runErr = errors.New("interna greška tijekom prijenosa")
		}
		m.finishJob(ctx, id, runErr)
		m.pump()
	}()

	job, ok := m.jobSnapshot(id)
	if !ok {
		runErr = errors.New("transfer nije pronađen")
		return
	}
	settings, err := m.settings.Get()
	if err != nil {
		runErr = err
		return
	}
	maxAttempts := 1 + settings.AutoRetryCount
	delay := time.Duration(settings.RetryDelaySeconds) * time.Second

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			runErr = err
			return
		}
		m.setAttempt(id, attempt)

		err := m.runAttempt(ctx, job, settings)
		if err == nil || errors.Is(err, remote.ErrSkipped) {
			runErr = err
			return
		}
		runErr = err
		if ctx.Err() != nil || attempt == maxAttempts || !remote.IsRetryable(err) {
			return
		}
		if err := waitRetryDelay(ctx, delay); err != nil {
			runErr = err
			return
		}
	}
}

func (m *Manager) Enable() {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return
	}
	m.generation++
	m.accepting = true
	m.paused = false
	jobs := append([]model.TransferJob(nil), m.jobs...)
	m.emitLocked(Event{Type: "state", Jobs: jobs, Paused: false})
	m.mu.Unlock()
	m.pump()
}

func (m *Manager) waitWorkers(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	done := make(chan struct{})
	go func() {
		m.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (m *Manager) DisableAndCancel(ctx context.Context, reason string) error {
	if strings.TrimSpace(reason) == "" {
		reason = "Otkazano prekidom veze"
	}
	m.mu.Lock()
	m.generation++
	m.accepting = false
	m.paused = true
	for i := range m.jobs {
		switch m.jobs[i].Status {
		case "queued":
			m.jobs[i].Status = "cancelled"
			m.jobs[i].Error = reason
			j := m.jobs[i]
			m.emitLocked(Event{Type: "job", Job: &j})
		case "running":
			if c := m.cancels[m.jobs[i].ID]; c != nil {
				c()
			}
		}
	}
	jobs := append([]model.TransferJob(nil), m.jobs...)
	m.emitLocked(Event{Type: "state", Jobs: jobs, Paused: true})
	m.mu.Unlock()
	return m.waitWorkers(ctx)
}

func (m *Manager) Shutdown(ctx context.Context) error {
	m.mu.Lock()
	m.generation++
	m.closed = true
	m.accepting = false
	m.paused = true
	for i := range m.jobs {
		if m.jobs[i].Status == "queued" {
			m.jobs[i].Status = "cancelled"
			m.jobs[i].Error = "Otkazano zatvaranjem aplikacije"
		}
	}
	for _, c := range m.cancels {
		c()
	}
	m.mu.Unlock()
	return m.waitWorkers(ctx)
}
