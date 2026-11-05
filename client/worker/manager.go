package worker

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"

	"github.com/charmbracelet/log"
	"github.com/krau/mygotg"
	"github.com/krau/mygotg/ext"
	"github.com/krau/mygotg/session"
	"github.com/rs/xid"

	"github.com/krau/SaveAny-Bot/common/utils/tgutil"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/database"
)

const (
	StateStopped  = "stopped"
	StateStarting = "starting"
	StateRunning  = "running"
	StateError    = "error"
)

type Status struct {
	State    string `json:"state"`
	Username string `json:"username,omitempty"`
	Error    string `json:"error,omitempty"`
}

type ClientRef struct {
	ID   uint
	Name string
	Ext  *ext.Context
}

type runtime struct {
	id       uint
	name     string
	session  string
	mu       sync.RWMutex
	state    string
	err      string
	username string
	client   *mygotg.Client
	ext      *ext.Context
	cancel   context.CancelFunc
	gen      uint64
}

type Manager struct {
	ctx     context.Context
	dataDir string
	mu      sync.RWMutex
	workers map[uint]*runtime
	order   []uint
	next    int
}

var manager *Manager

// Init creates the optional helper-bot manager. A manager with no workers is a
// no-op, preserving the zero-config single-bot setup.
func Init(ctx context.Context, dataDir string) {
	m := &Manager{ctx: ctx, dataDir: dataDir, workers: make(map[uint]*runtime)}
	manager = m
	if err := m.reload(ctx); err != nil {
		log.Errorf("Failed to load helper bots: %v", err)
	}
}

func (m *Manager) reload(ctx context.Context) error {
	rows, err := database.ListBotWorkers(ctx)
	if err != nil {
		return err
	}
	m.mu.Lock()
	old := m.workers
	m.workers = make(map[uint]*runtime, len(rows))
	m.order = m.order[:0]
	m.next = 0
	m.mu.Unlock()
	for _, rt := range old {
		stopRuntime(rt)
	}
	for _, row := range rows {
		m.mu.Lock()
		rt := &runtime{id: row.ID, name: row.Name, session: row.SessionPath, state: StateStopped}
		m.workers[row.ID] = rt
		m.order = append(m.order, row.ID)
		m.mu.Unlock()
		if row.Enabled {
			m.start(ctx, rt, row.Token)
		}
	}
	return nil
}

func (m *Manager) start(ctx context.Context, rt *runtime, token string) {
	if token == "" {
		rt.mu.Lock()
		rt.state, rt.err = StateError, "token is empty"
		rt.mu.Unlock()
		return
	}
	sessionPath := rt.session
	if sessionPath == "" {
		sessionPath = NewSessionPath(m.dataDir)
		if err := database.UpdateBotWorker(ctx, rt.id, map[string]any{"session_path": sessionPath}); err != nil {
			rt.mu.Lock()
			rt.state, rt.err = StateError, err.Error()
			rt.mu.Unlock()
			return
		}
	}
	if err := os.MkdirAll(filepath.Dir(sessionPath), 0o755); err != nil {
		rt.mu.Lock()
		rt.state, rt.err = StateError, err.Error()
		rt.mu.Unlock()
		return
	}

	runCtx, cancel := context.WithCancel(ctx)
	rt.mu.Lock()
	rt.session = sessionPath
	rt.gen++
	gen := rt.gen
	rt.cancel = cancel
	rt.state, rt.err, rt.username = StateStarting, "", ""
	rt.mu.Unlock()

	go func() {
		resolver, err := tgutil.NewConfigProxyResolver()
		if err != nil {
			m.finishStart(rt, gen, cancel, nil, err, "")
			return
		}
		client, err := mygotg.NewClient(
			config.C().Telegram.AppID,
			config.C().Telegram.AppHash,
			mygotg.ClientTypeBot(token),
			&mygotg.ClientOpts{
				Session:          session.SqlSession(database.GetDialect(sessionPath)),
				Context:          runCtx,
				NoUpdates:        true,
				DisableCopyright: true,
				Resolver:         resolver,
				MaxRetries:       config.C().Telegram.RpcRetry,
			},
		)
		if err != nil {
			m.finishStart(rt, gen, cancel, nil, err, "")
			return
		}
		username := ""
		if client.Self != nil {
			username = client.Self.Username
		}
		m.finishStart(rt, gen, cancel, client, nil, username)
	}()
}

func (m *Manager) finishStart(rt *runtime, gen uint64, cancel context.CancelFunc, client *mygotg.Client, err error, username string) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	if rt.gen != gen {
		if client != nil {
			client.Stop()
		}
		cancel()
		return
	}
	if err != nil {
		rt.state, rt.err = StateError, err.Error()
		rt.cancel = nil
		cancel()
		return
	}
	rt.client = client
	rt.ext = client.CreateContext()
	rt.state = StateRunning
	rt.err = ""
	rt.username = username
}

// StartWorker starts one persisted worker and returns its current status.
func StartWorker(ctx context.Context, id uint) (Status, error) {
	if manager == nil {
		return Status{}, errors.New("helper bot manager is not initialized")
	}
	row, err := database.GetBotWorker(ctx, id)
	if err != nil {
		return Status{}, err
	}
	if !row.Enabled {
		return Status{}, errors.New("helper bot is disabled")
	}
	manager.mu.Lock()
	rt := manager.workers[id]
	if rt == nil {
		rt = &runtime{id: row.ID, name: row.Name, session: row.SessionPath, state: StateStopped}
		manager.workers[id] = rt
		manager.order = append(manager.order, id)
	}
	manager.mu.Unlock()
	rt.mu.RLock()
	state := rt.state
	rt.mu.RUnlock()
	if state == StateRunning || state == StateStarting {
		return StatusFor(id), nil
	}
	rt.mu.Lock()
	rt.name = row.Name
	if row.SessionPath != "" {
		rt.session = row.SessionPath
	}
	rt.mu.Unlock()
	lifecycleCtx := manager.ctx
	if lifecycleCtx == nil {
		lifecycleCtx = ctx
	}
	manager.start(lifecycleCtx, rt, row.Token)
	return StatusFor(id), nil
}

// StopWorker stops one worker while keeping its persisted row.
func StopWorker(id uint) {
	if manager == nil {
		return
	}
	manager.mu.RLock()
	rt := manager.workers[id]
	manager.mu.RUnlock()
	if rt != nil {
		stopRuntime(rt)
	}
}

func StopAll() {
	if manager == nil {
		return
	}
	manager.mu.RLock()
	workers := make([]*runtime, 0, len(manager.workers))
	for _, rt := range manager.workers {
		workers = append(workers, rt)
	}
	manager.mu.RUnlock()
	for _, rt := range workers {
		stopRuntime(rt)
	}
}

func stopRuntime(rt *runtime) {
	rt.mu.Lock()
	client := rt.client
	rt.client = nil
	rt.ext = nil
	rt.state, rt.err, rt.username = StateStopped, "", ""
	rt.gen++
	cancel := rt.cancel
	rt.cancel = nil
	rt.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if client != nil {
		client.Stop()
	}
}

// UploadExts returns running helper contexts in round-robin order. Callers can
// skip helpers that cannot access the intended target chat.
func UploadExts() []*ext.Context {
	if manager == nil {
		return nil
	}
	manager.mu.Lock()
	defer manager.mu.Unlock()
	if len(manager.order) == 0 {
		return nil
	}
	exts := make([]*ext.Context, 0, len(manager.order))
	start := manager.next
	first := -1
	for i := range manager.order {
		idx := (start + i) % len(manager.order)
		rt := manager.workers[manager.order[idx]]
		if rt == nil {
			continue
		}
		rt.mu.RLock()
		state, extCtx := rt.state, rt.ext
		rt.mu.RUnlock()
		if state == StateRunning && extCtx != nil {
			if first == -1 {
				first = idx
			}
			exts = append(exts, extCtx)
		}
	}
	if first >= 0 {
		manager.next = (first + 1) % len(manager.order)
	}
	return exts
}

// UploadExt returns the next running helper context for one Telegram upload,
// or nil when no helper is configured.
func UploadExt() *ext.Context {
	exts := UploadExts()
	if len(exts) > 0 {
		return exts[0]
	}
	return nil
}

// RunningClients returns a snapshot of running helpers for download routing.
func RunningClients() []ClientRef {
	if manager == nil {
		return nil
	}
	manager.mu.RLock()
	defer manager.mu.RUnlock()
	refs := make([]ClientRef, 0, len(manager.order))
	for _, id := range manager.order {
		rt := manager.workers[id]
		if rt == nil {
			continue
		}
		rt.mu.RLock()
		if rt.state == StateRunning && rt.ext != nil {
			refs = append(refs, ClientRef{ID: rt.id, Name: rt.name, Ext: rt.ext})
		}
		rt.mu.RUnlock()
	}
	return refs
}

// StatusFor returns the runtime status for a worker id.
func StatusFor(id uint) Status {
	if manager == nil {
		return Status{State: StateStopped}
	}
	manager.mu.RLock()
	rt := manager.workers[id]
	manager.mu.RUnlock()
	if rt == nil {
		return Status{State: StateStopped}
	}
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	return Status{State: rt.state, Username: rt.username, Error: rt.err}
}

// NewSessionPath returns a unique session database path under data/workers.
func NewSessionPath(dataDir string) string {
	return filepath.Join(dataDir, "workers", xid.New().String()+".db")
}

// RemoveSession deletes a worker session database and its SQLite side files.
func RemoveSession(path string) error {
	if path == "" {
		return nil
	}
	for _, suffix := range []string{"", "-wal", "-shm"} {
		if err := os.Remove(path + suffix); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}
