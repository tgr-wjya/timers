package daemon

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
	"timers/internal/ipc"
	"timers/internal/notify"
	"timers/internal/pomodoro"
	"timers/internal/sound"
	"timers/internal/storage"
	"timers/internal/timer"
)

type Daemon struct {
	socketPath string
	dbPath     string
	engine     *timer.Engine
	store      *storage.Store
	server     *ipc.Server
	stopChan   chan struct{}
	stopOnce   sync.Once
	mu         sync.Mutex
	activePomo *pomodoro.Cycle
}

func NewDaemon(socketPath, dbPath string) (*Daemon, error) {
	store, err := storage.NewStore(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize store: %w", err)
	}

	d := &Daemon{
		socketPath: socketPath,
		dbPath:     dbPath,
		store:      store,
		stopChan:   make(chan struct{}),
	}

	// Completion callback from engine
	cb := func(tm *timer.Timer, event string) {
		if event == "completed" {
			d.handleTimerCompleted(tm)
		}
		// Persist update in database
		_ = d.store.UpdateTimer(tm)
	}

	d.engine = timer.NewEngine(cb)

	// Set nextID to avoid ID collision with previous sessions
	if maxID, err := d.store.GetMaxID(); err == nil && maxID > 0 {
		d.engine.SetNextID(maxID + 1)
	}

	// Recover active timers from DB if any
	active, err := d.store.GetActiveTimers()
	if err == nil {
		now := time.Now()
		var alive []*timer.Timer
		for _, a := range active {
			if a.Status == timer.StatusRunning && !now.Before(a.TargetEnd) {
				// Expired while daemon was off
				a.Status = timer.StatusCompleted
				ended := a.TargetEnd
				a.EndedAt = &ended
				_ = d.store.UpdateTimer(a)
			} else {
				alive = append(alive, a)
			}
		}
		if len(alive) > 0 {
			d.engine.Restore(alive)
		}
	}

	server, err := ipc.NewServer(socketPath, d.handleRequest)
	if err != nil {
		_ = store.Close()
		return nil, fmt.Errorf("failed to start IPC server: %w", err)
	}
	d.server = server

	return d, nil
}

func (d *Daemon) Start() error {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-sigChan:
			d.Stop()
			return nil
		case <-d.stopChan:
			return nil
		case now := <-ticker.C:
			d.engine.Tick(now)
		}
	}
}

func (d *Daemon) handleTimerCompleted(tm *timer.Timer) {
	// Audio chime (unless muted)
	if !tm.Mute {
		if tm.Category == timer.CategoryPomodoroBreak {
			sound.Play(sound.SoundComplete)
		} else {
			sound.Play(sound.SoundAlarm)
		}
	}

	// Notification
	title := fmt.Sprintf("Timer Finished: %s", tm.Title)
	msg := fmt.Sprintf("Duration: %s completed at %s.", tm.Duration.String(), time.Now().Format("15:04:05"))
	if tm.Category == timer.CategoryPomodoroWork {
		title = "Pomodoro: Focus Block Finished!"
		msg = fmt.Sprintf("Time for a break! Completed 25m session on '%s'.", tm.Title)
	} else if tm.Category == timer.CategoryPomodoroBreak {
		title = "Pomodoro: Break Ended!"
		msg = "Ready to start the next focus block?"
	}

	notify.Send(title, msg, true)
}

func (d *Daemon) handleRequest(req ipc.Request) ipc.Response {
	d.mu.Lock()
	defer d.mu.Unlock()

	switch req.Action {
	case "ping":
		return ipc.Response{Status: "ok", Message: "pong"}

	case "start":
		dur := time.Duration(req.DurationSec) * time.Second
		cat := timer.Category(req.Category)
		if cat == "" {
			cat = timer.CategoryGeneral
		}
		tm := d.engine.Start(req.Title, dur, cat, req.Tags)
		tm.Mute = req.Mute
		_ = d.store.SaveTimer(tm)
		return ipc.Response{Status: "ok", Timer: tm}

	case "list":
		timers := d.engine.List()
		return ipc.Response{Status: "ok", Timers: timers}

	case "pause":
		tm, err := d.engine.Pause(req.ID)
		if err != nil {
			return ipc.Response{Status: "error", Message: err.Error()}
		}
		_ = d.store.UpdateTimer(tm)
		return ipc.Response{Status: "ok", Timer: tm}

	case "resume":
		tm, err := d.engine.Resume(req.ID)
		if err != nil {
			return ipc.Response{Status: "error", Message: err.Error()}
		}
		_ = d.store.UpdateTimer(tm)
		return ipc.Response{Status: "ok", Timer: tm}

	case "cancel":
		tm, err := d.engine.Cancel(req.ID)
		if err != nil {
			return ipc.Response{Status: "error", Message: err.Error()}
		}
		_ = d.store.UpdateTimer(tm)
		return ipc.Response{Status: "ok", Timer: tm}

	case "cancel_all":
		timers := d.engine.List()
		for _, t := range timers {
			if t.Status == timer.StatusRunning || t.Status == timer.StatusPaused {
				cancelled, err := d.engine.Cancel(t.ID)
				if err == nil {
					_ = d.store.UpdateTimer(cancelled)
				}
			}
		}
		return ipc.Response{Status: "ok", Message: "all active timers cancelled"}

	case "pomo_start":
		cfg := pomodoro.DefaultConfig()
		if req.DurationSec > 0 {
			cfg.WorkDuration = time.Duration(req.DurationSec) * time.Second
		}
		cycle := pomodoro.NewCycle(req.Title, cfg)
		d.activePomo = cycle
		_ = d.store.SavePomodoroCycle(cycle)

		// Start first work timer
		tm := d.engine.Start(
			fmt.Sprintf("%s (Round %d/%d)", cycle.TaskName, cycle.Round, cycle.Config.TotalRounds),
			cycle.Config.WorkDuration,
			timer.CategoryPomodoroWork,
			req.Tags,
		)
		_ = d.store.SaveTimer(tm)
		return ipc.Response{Status: "ok", Timer: tm}

	case "pomo_next":
		if d.activePomo == nil {
			return ipc.Response{Status: "error", Message: "no active pomodoro cycle"}
		}
		stage := d.activePomo.NextStage()
		_ = d.store.SavePomodoroCycle(d.activePomo)

		if stage == pomodoro.StageCompleted {
			d.activePomo = nil
			return ipc.Response{Status: "ok", Message: "Pomodoro cycle completed!"}
		}

		dur := d.activePomo.DurationForStage(stage)
		cat := timer.CategoryPomodoroBreak
		title := fmt.Sprintf("%s Break (Round %d/%d)", d.activePomo.TaskName, d.activePomo.Round, d.activePomo.Config.TotalRounds)
		if stage == pomodoro.StageWork {
			cat = timer.CategoryPomodoroWork
			title = fmt.Sprintf("%s (Round %d/%d)", d.activePomo.TaskName, d.activePomo.Round, d.activePomo.Config.TotalRounds)
		}

		tm := d.engine.Start(title, dur, cat, nil)
		_ = d.store.SaveTimer(tm)
		return ipc.Response{Status: "ok", Timer: tm}

	case "stop":
		go func() {
			time.Sleep(50 * time.Millisecond)
			d.Stop()
		}()
		return ipc.Response{Status: "ok", Message: "daemon shutting down"}

	default:
		return ipc.Response{Status: "error", Message: fmt.Sprintf("unknown action: %s", req.Action)}
	}
}

func (d *Daemon) Stop() {
	d.stopOnce.Do(func() {
		close(d.stopChan)
		if d.server != nil {
			_ = d.server.Close()
		}
		if d.store != nil {
			_ = d.store.Close()
		}
		if d.engine != nil {
			d.engine.Stop()
		}
	})
}
