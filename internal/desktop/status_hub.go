package desktop

import (
	"sync"
	"time"
)

// StatusEmitter sends project status updates to the UI.
type StatusEmitter func(ProjectStatus)

// StatusHub polls pinned projects in parallel with capped concurrency.
type StatusHub struct {
	mu       sync.Mutex
	paths    []string
	aliases  map[string]string
	active   string
	cache    map[string]ProjectStatus
	emit     StatusEmitter
	stopCh   chan struct{}
	running  bool
	paused   bool
	tick     int
	interval time.Duration
}

// NewStatusHub creates a hub. Call Start after setting emitter.
func NewStatusHub(emit StatusEmitter) *StatusHub {
	return &StatusHub{
		aliases:  map[string]string{},
		cache:    map[string]ProjectStatus{},
		emit:     emit,
		interval: 15 * time.Second,
	}
}

// Start begins background polling.
func (h *StatusHub) Start() {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.running {
		return
	}
	h.stopCh = make(chan struct{})
	h.running = true
	go h.loop(h.stopCh)
}

// Stop ends background polling.
func (h *StatusHub) Stop() {
	h.mu.Lock()
	defer h.mu.Unlock()
	if !h.running {
		return
	}
	close(h.stopCh)
	h.running = false
}

// SetPinned replaces watched paths (max MaxPinned) and optional aliases.
func (h *StatusHub) SetPinned(pinned []PinnedProject, active string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.paths = h.paths[:0]
	h.aliases = map[string]string{}
	for i, p := range pinned {
		if i >= MaxPinned {
			break
		}
		h.paths = append(h.paths, p.Path)
		if p.Alias != "" {
			h.aliases[p.Path] = p.Alias
		}
	}
	h.active = active
}

// SetActive marks the focused project (gets PR checks more often).
func (h *StatusHub) SetActive(path string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.active = path
}

// SetPaused suspends background polling (window hidden / app inactive).
func (h *StatusHub) SetPaused(paused bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.paused = paused
}

// Paused reports whether background polling is suspended.
func (h *StatusHub) Paused() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.paused
}

// Snapshot returns cached statuses in pinned order.
func (h *StatusHub) Snapshot() []ProjectStatus {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]ProjectStatus, 0, len(h.paths))
	for _, path := range h.paths {
		st, ok := h.cache[path]
		if !ok {
			st = ProjectStatus{Path: path, RepoName: path}
		}
		st.Active = samePath(path, h.active)
		if alias := h.aliases[path]; alias != "" {
			st.Alias = alias
		}
		out = append(out, st)
	}
	return out
}

// CacheStatus upserts one project status without triggering a poll.
func (h *StatusHub) CacheStatus(st ProjectStatus) {
	h.mu.Lock()
	defer h.mu.Unlock()
	key := st.Path
	for _, p := range h.paths {
		if samePath(p, st.Path) {
			key = p
			break
		}
	}
	st.Path = key
	if alias := h.aliases[key]; alias != "" {
		st.Alias = alias
	}
	h.cache[key] = st
}

// RefreshNow runs one poll cycle synchronously (light, no PR except active).
func (h *StatusHub) RefreshNow() []ProjectStatus {
	h.poll(false)
	return h.Snapshot()
}

func (h *StatusHub) loop(stop <-chan struct{}) {
	// immediate first pass
	h.poll(true)
	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			h.mu.Lock()
			paused := h.paused
			h.mu.Unlock()
			if paused {
				continue
			}
			h.poll(false)
		}
	}
}

func (h *StatusHub) poll(forcePR bool) {
	h.mu.Lock()
	if h.paused && !forcePR {
		h.mu.Unlock()
		return
	}
	paths := append([]string{}, h.paths...)
	active := h.active
	aliases := map[string]string{}
	for k, v := range h.aliases {
		aliases[k] = v
	}
	h.tick++
	tick := h.tick
	h.mu.Unlock()

	if len(paths) == 0 {
		return
	}

	includePRActive := forcePR || tick%4 == 0 // ~60s with 15s interval

	var wg sync.WaitGroup
	sem := make(chan struct{}, 3) // cap parallelism
	results := make([]ProjectStatus, len(paths))

	for i, path := range paths {
		wg.Add(1)
		go func(i int, path string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			withPR := includePRActive && samePath(path, active)
			st := LoadProjectStatus(path, withPR)
			if alias := aliases[path]; alias != "" {
				st.Alias = alias
			}
			st.Active = samePath(path, active)
			results[i] = st
		}(i, path)
	}
	wg.Wait()

	h.mu.Lock()
	for _, st := range results {
		// Preserve previous PR/CI info if this tick skipped gh.
		if !includePRActive || !samePath(st.Path, active) {
			if prev, ok := h.cache[st.Path]; ok && st.Error == "" {
				if prev.HasOpenPR && !st.HasOpenPR {
					st.HasOpenPR = prev.HasOpenPR
					st.PRTitle = prev.PRTitle
				}
				if prev.CILabel != "" && st.CILabel == "" {
					st.CIState = prev.CIState
					st.CILabel = prev.CILabel
					st.CIFromCache = prev.CIFromCache
					st.CIHost = prev.CIHost
				}
			}
		}
		h.cache[st.Path] = st
	}
	emit := h.emit
	h.mu.Unlock()

	if emit != nil {
		for _, st := range results {
			emit(st)
		}
	}
}
