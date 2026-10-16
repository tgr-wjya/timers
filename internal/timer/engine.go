package timer

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

type CallbackFunc func(t *Timer, event string)

type Engine struct {
	mu       sync.RWMutex
	timers   map[int]*Timer
	nextID   int
	callback CallbackFunc
	stopOnce sync.Once
	stopChan chan struct{}
}

func NewEngine(cb CallbackFunc) *Engine {
	e := &Engine{
		timers:   make(map[int]*Timer),
		nextID:   1,
		callback: cb,
		stopChan: make(chan struct{}),
	}
	return e
}

func (e *Engine) Start(title string, duration time.Duration, category Category, tags []string) *Timer {
	e.mu.Lock()
	defer e.mu.Unlock()

	now := time.Now()
	if category == "" {
		category = CategoryGeneral
	}
	if title == "" {
		title = fmt.Sprintf("Timer #%d", e.nextID)
	}

	t := &Timer{
		ID:        e.nextID,
		Title:     title,
		Category:  category,
		Duration:  duration,
		StartedAt: now,
		TargetEnd: now.Add(duration),
		Status:    StatusRunning,
		Tags:      tags,
	}

	e.timers[t.ID] = t
	e.nextID++

	if e.callback != nil {
		go e.callback(t, "started")
	}

	return t
}

func (e *Engine) Pause(id int) (*Timer, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	t, ok := e.timers[id]
	if !ok {
		return nil, fmt.Errorf("timer #%d not found", id)
	}
	if t.Status != StatusRunning {
		return nil, fmt.Errorf("timer #%d is not running (status: %s)", id, t.Status)
	}

	now := time.Now()
	rem := t.TargetEnd.Sub(now)
	if rem < 0 {
		rem = 0
	}
	t.RemainingWhenPaused = rem
	t.Status = StatusPaused

	if e.callback != nil {
		go e.callback(t, "paused")
	}

	return t, nil
}

func (e *Engine) Resume(id int) (*Timer, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	t, ok := e.timers[id]
	if !ok {
		return nil, fmt.Errorf("timer #%d not found", id)
	}
	if t.Status != StatusPaused {
		return nil, fmt.Errorf("timer #%d is not paused (status: %s)", id, t.Status)
	}

	now := time.Now()
	t.TargetEnd = now.Add(t.RemainingWhenPaused)
	t.Status = StatusRunning
	t.RemainingWhenPaused = 0

	if e.callback != nil {
		go e.callback(t, "resumed")
	}

	return t, nil
}

func (e *Engine) Cancel(id int) (*Timer, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	t, ok := e.timers[id]
	if !ok {
		return nil, fmt.Errorf("timer #%d not found", id)
	}
	if t.Status == StatusCompleted || t.Status == StatusCancelled {
		return nil, fmt.Errorf("timer #%d is already %s", id, t.Status)
	}

	now := time.Now()
	t.Status = StatusCancelled
	t.EndedAt = &now

	if e.callback != nil {
		go e.callback(t, "cancelled")
	}

	return t, nil
}

func (e *Engine) Get(id int) (*Timer, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	t, ok := e.timers[id]
	if !ok {
		return nil, false
	}
	// Return a shallow copy
	cp := *t
	return &cp, true
}

func (e *Engine) List() []*Timer {
	e.mu.RLock()
	defer e.mu.RUnlock()

	list := make([]*Timer, 0, len(e.timers))
	for _, t := range e.timers {
		cp := *t
		list = append(list, &cp)
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].ID < list[j].ID
	})

	return list
}

func (e *Engine) Tick(now time.Time) []*Timer {
	e.mu.Lock()
	var expired []*Timer
	for _, t := range e.timers {
		if t.Status == StatusRunning && !now.Before(t.TargetEnd) {
			t.Status = StatusCompleted
			t.EndedAt = &now
			expired = append(expired, t)
		}
	}
	e.mu.Unlock()

	if e.callback != nil {
		for _, t := range expired {
			e.callback(t, "completed")
		}
	}

	return expired
}

func (e *Engine) Stop() {
	e.stopOnce.Do(func() {
		close(e.stopChan)
	})
}
