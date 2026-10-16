# Design Spec: Universal Timer CLI (`timer`)

**Date:** 2026-10-16  
**Status:** Approved by User  
**Target Platform:** Linux (Nobara / Kitty primary), macOS, Windows; all UTF-8 terminal emulators

---

## 1. Overview & Goals

`timer` is a fast, terminal-agnostic, cross-platform CLI timer and productivity tool written in Go. It provides human-friendly duration syntax (e.g. `timer 1h30m`, `timer 45s`), concurrent named timers managed by an auto-spawning background daemon, Pomodoro workflows with stage progression, native desktop notifications, audio chimes, and persistent SQLite session analytics.

### Key Goals
- **Human Duration Parsing:** Accepts compound formats like `1h30m`, `45s`, `2h`, `10m30s`, `1.5h` with helpful error messages on ambiguous inputs.
- **Concurrent & Non-Blocking:** Timers run in a detached background daemon; starting a timer immediately frees the shell prompt unless watch/foreground mode is requested.
- **Terminal Agnostic (Style 4 UI):** Features a sleek minimalist interface—Unicode clock glyphs (`○ ◔ ◑ ◕ ●`), slim continuous filament gauges (`━━━╸┈┈┈`), and universal XTerm OSC 2 window/tab title updates (`\033]2;...`). Works across Kitty, Alacritty, WezTerm, GNOME Terminal, Windows Terminal, and macOS Terminal.
- **Cross-Platform Audio & Alerts:** Fires native notifications (`notify-send` on Linux, `osascript` on macOS, Toast on Windows) and low-latency audio chimes (`pw-play`/`paplay`/`aplay` on Linux, `afplay` on macOS, PowerShell/WinMM on Windows, plus an embedded fallback chime in the binary).
- **Session Logging & Analytics:** Pure Go SQLite engine (`modernc.org/sqlite`, zero CGO) storing completed sessions in standard XDG directories with query commands (`timer log`, `timer stats`).
- **Zero-Friction Daemon:** Single binary `timer` that automatically spawns its background daemon on demand if not already running.

---

## 2. CLI Command Specification

The binary is invoked as `timer <command>` or with a duration directly as shorthand.

### 2.1 Starting Timers
```bash
# Direct shorthand (starts in background, returns immediately)
timer 25m "Write Spec"
timer 1h30m
timer 45s "Tea"

# Foreground / Watch mode (in-place live progress until finish or Ctrl+C)
timer 25m "Deep Work" -f
timer watch 1
```

**Output on start (Background):**
```
Timer #1 'Write Spec' started [25m0s] — alerts at 22:35
```

**Output in Foreground / Watch Mode (`-f` or `timer watch <id>`):**
```
  ◐ 18:42  ━━━━━━━━━━━━╸┈┈┈┈┈┈┈┈  Write Spec (74%)
```
*Note:* Also updates the terminal window/tab title via OSC 2: `\033]2;◐ 18:42 Write Spec\007`. On completion, it displays `✔ 00:00 Timer Finished!` before resetting the terminal title. Detaching (`Ctrl+C` or `q`) leaves the timer running in the background daemon.

### 2.2 Managing Active Timers
- `timer list` (alias: `timer ls`):
  Lists all active, paused, and recently finished timers in an aligned table:
  ```
  ID  TITLE        REMAINING   ELAPSED   STATUS   TARGET END
  #1  Write Spec   18m 42s     06m 18s   Running  22:35:10
  #2  Tea          00m 15s     00m 30s   Running  22:15:45
  ```
- `timer pause <id>`: Freezes timer countdown.
- `timer resume <id>`: Resumes timer from paused state without drift.
- `timer cancel <id|all>`: Cancels and removes the active timer.

### 2.3 Pomodoro Workflow
```bash
timer pomodoro [task-name]
timer pomo "Refactor Parser"
timer pomo --work 30m --short-break 5m --long-break 20m --rounds 4
```
- Starts a 25m focus block.
- Upon completion:
  1. Plays a mellow transition chime (`complete.oga`) and dispatches desktop notification: `"Pomodoro: Work session complete! Ready for 5m break?"`.
  2. If running interactive CLI, prompts user or allows running `timer break` to start the break.
  3. Tracks cycles (Round 1 to 4) and triggers a 15m long break after 4 work rounds.

### 2.4 History & Analytics
- `timer log` (alias: `timer history`):
  ```
  ID   DATE         TIME    DURATION  TITLE             STATUS
  #14  2026-10-16   21:30   25m 00s   Refactor parser   Completed
  #13  2026-10-16   20:45   10m 00s   Quick sync        Completed
  #12  2026-10-16   20:10   45m 00s   Draft proposal    Cancelled
  ```
  Flags: `--today`, `--yesterday`, `--week`, `--since <duration>`, `--limit <n>`.
- `timer stats`:
  Displays total focus time, pomodoros completed, completion percentage, and task breakdown.

### 2.5 Daemon Management
- `timer daemon`: Explicitly starts the daemon in foreground (used internally by auto-spawner).
- `timer daemon status`: Checks if the daemon process and socket are active.
- `timer daemon stop`: Sends shutdown signal to daemon.

---

## 3. Architecture & Components

```
                ┌──────────────────────────────────────────────┐
                │                  CLI Client                  │
                │  (Duration parser, commands, watch renderer) │
                └──────────────────────┬───────────────────────┘
                                       │ Unix Domain Socket
                                       │ (Line-delimited JSON IPC)
                                       ▼
                ┌──────────────────────────────────────────────┐
                │              Background Daemon               │
                │  Auto-spawning, single-instance supervisor   │
                └──────┬───────────────┬───────────────┬───────┘
                       │               │               │
                       ▼               ▼               ▼
               ┌──────────────┐ ┌─────────────┐ ┌──────────────┐
               │ Timer Engine │ │ SQLite Store│ │ Alert Engine │
               │ Scheduler &  │ │ WAL mode    │ │ Notifications│
               │ State Machine│ │ (pure Go)   │ │ & Audio Chime│
               └──────────────┘ └─────────────┘ └──────────────┘
```

### 3.1 Daemon Lifecycle & IPC
1. **Socket Endpoint:**
   - Linux: `$XDG_RUNTIME_DIR/timer/timer.sock` (fallback `/tmp/timer-$UID/timer.sock`).
   - macOS: `~/Library/Caches/timer/timer.sock` or `/tmp/timer-$UID.sock`.
   - Windows: Named Pipe `\\.\pipe\timer-$USERNAME` or loopback TCP `127.0.0.1:0` with port lockfile.
2. **Auto-Spawn Protocol:**
   - Client dials socket with a 50ms timeout.
   - If missing/unreachable, client launches `timer daemon` detached (`os.StartProcess` with `Setsid: true`).
   - Client retries connection up to 5 times with exponential backoff (typically connected within 30-50ms).
   - If socket is stale (daemon dead), the client deletes the orphan socket file and spawns a fresh daemon.
3. **IPC Messages (JSON Lines):**
   - Client -> Daemon: `{"action": "start", "duration_sec": 1500, "title": "Focus", "category": "pomodoro_work"}`
   - Daemon -> Client: `{"status": "ok", "timer": {"id": 1, "title": "Focus", "remaining_sec": 1500, "state": "running"}}`
   - Watch Stream: Daemon emits 1Hz event stream: `{"event": "tick", "id": 1, "remaining_sec": 1120, "progress": 0.25}`.

### 3.2 Timer Scheduling (Zero Drift)
- Timers store an absolute `TargetEnd = StartedAt + Duration`.
- Suspending/waking the computer does not cause drift.
- A 1Hz master ticker checks for `Now() >= TargetEnd`.
- When paused: `Remaining = TargetEnd - Now()`. When resumed: `TargetEnd = Now() + Remaining`.
- Concurrency: `sync.RWMutex` protects active timers in memory. Expirations trigger asynchronous alert dispatch so the ticker loop is never blocked.

---

## 4. Notifications & Audio Subsystem

### 4.1 Audio Chime Dispatcher
Abstracted behind `type AudioPlayer interface { PlayChime(soundType SoundKind) error }`:
- **Linux:** Probes binaries in order:
  1. `pw-play` (PipeWire native)
  2. `paplay` (PulseAudio)
  3. `aplay` (ALSA fallback)
  - Default sound paths:
    - Standard finish: `/usr/share/sounds/freedesktop/stereo/alarm-clock-elapsed.oga`
    - Phase/break: `/usr/share/sounds/freedesktop/stereo/complete.oga`
- **macOS:** Calls `/usr/bin/afplay /System/Library/Sounds/Glass.aiff`.
- **Windows:** Calls PowerShell `[System.Media.SystemSounds]::Exclamation.Play()` or plays WAV via Windows multimedia API.
- **Embedded Fallback:** Includes embedded chime WAV via `//go:embed` to ensure sound works even on minimal systems lacking desktop sound themes.
- **Terminal Bell:** Emits ASCII BEL (`\a`) as a universal fallback.

### 4.2 Desktop Notifications
Abstracted behind `type Notifier interface { Notify(title, message string, urgency string) error }`:
- **Linux:** Executes `notify-send --app-name="timer" --urgency=critical --icon=alarm-symbolic <title> <message>`.
- **macOS:** Executes `osascript -e 'display notification "<message>" with title "<title>"'`.
- **Windows:** Emits toast notification via PowerShell WinRT or BurntToast script.

---

## 5. Storage & Analytics (SQLite)

- **Engine:** `modernc.org/sqlite` (Pure Go, `CGO_ENABLED=0`, compiles anywhere).
- **Location:**
  - Linux: `~/.local/share/timer/timers.db`
  - macOS: `~/Library/Application Support/timer/timers.db`
  - Windows: `%LOCALAPPDATA%\timer\timers.db`
- **Database Configuration:** WAL mode (`PRAGMA journal_mode = WAL;`) for fast concurrent reads and durable atomic writes.

### Schema
```sql
CREATE TABLE IF NOT EXISTS timers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL,
    category TEXT NOT NULL DEFAULT 'general',
    duration_sec INTEGER NOT NULL,
    started_at DATETIME NOT NULL,
    target_end DATETIME NOT NULL,
    ended_at DATETIME NULL,
    status TEXT NOT NULL, -- 'running', 'paused', 'completed', 'cancelled'
    remaining_sec INTEGER NOT NULL DEFAULT 0,
    pomodoro_cycle_id TEXT NULL,
    tags TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS pomodoro_cycles (
    id TEXT PRIMARY KEY,
    task_name TEXT NOT NULL,
    current_round INTEGER NOT NULL DEFAULT 1,
    total_rounds INTEGER NOT NULL DEFAULT 4,
    state TEXT NOT NULL, -- 'work', 'short_break', 'long_break', 'completed'
    started_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);
```

---

## 6. Terminal UI Design (Style 4)

- **Renderer:** ANSI VT100 compliant, compatible with all UTF-8 terminals.
- **Layout:**
  - Dynamic Clock Glyph: 8 phases (`○`, `◔`, `◑`, `◕`, `●`) rotating or filling smoothly as a fraction of time elapsed.
  - Filament Gauge: Clean continuous track `━━━━━━━━━━━━╸┈┈┈┈┈┈┈┈┈┈┈┈` with percentage and countdown.
  - Window Title (OSC 2): `\033]2;◐ 18:42 Write Spec\007`.
  - Non-TTY fallback: If stdout is redirected or non-interactive, suppresses ANSI cursor escapes and OSC sequences.

---

## 7. Package Layout

```
timers/
├── cmd/
│   └── timer/
│       └── main.go              # CLI entry point, flag dispatch
├── internal/
│   ├── cli/                     # User-facing commands (start, list, cancel, pause, pomo, log, stats)
│   ├── daemon/                  # Background daemon server, socket listener, auto-spawner
│   ├── timer/                   # Core timer model, absolute-time scheduler, ticker loop
│   ├── pomodoro/                # Pomodoro cycle state machine
│   ├── storage/                 # Pure Go SQLite repository, WAL mode, migrations, analytics
│   ├── ipc/                     # Socket client/server, message framing, ping/pong
│   ├── notify/                  # Cross-platform desktop notification (Linux, macOS, Windows)
│   ├── sound/                   # Cross-platform audio dispatcher (PipeWire, Pulse, macOS, Win)
│   └── ui/                      # Terminal progress rendering (Style 4, OSC title, fallback)
├── assets/
│   └── sounds/                  # Embedded fallback chime (//go:embed)
├── go.mod
├── go.sum
└── README.md
```

---

## 8. Testing & Error Handling Strategy

### 8.1 Unit Tests
- `internal/timer`: Duration parsing edge cases (`1h30m`, `45s`, `1.5h`, invalid formats), scheduler drift math, pause/resume remaining duration.
- `internal/pomodoro`: Round cycle transitions (Work $\to$ Short Break $\to$ Work $\to$ Long Break $\to$ Reset).
- `internal/storage`: SQLite in-memory tests for session creation, updates, log filtering (`--today`, `--since 7d`), and aggregation stats.
- `internal/ipc`: JSON lines serialization, round-trip command execution.

### 8.2 Integration Tests
- Ephemeral daemon integration test: spawns daemon on a test socket, creates a fast timer (`200ms`), waits for completion event, verifies record is written to SQLite, and shuts down daemon cleanly.
