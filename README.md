# timers

### 16 october 2026

> blazingly fast zero-drift cli timer with desktop alerts and pomodoro

i wanted a fast, lightweight, and reliable cli timer that gets out of my way. no electron bloat, no clunky web ui, and no drifting timers when my machine sleeps or when the cpu gets slammed.

built with go, running as a detached background daemon over unix domain sockets with pure-go sqlite persistence. works out of the box on linux (specifically tuned for my nobara setup with pipewire and kitty, but terminal-agnostic so any ansi terminal works) and compiles cleanly on macos and windows without any cgo nonsense.

## explore

- [timers](#timers)
  - [what it does](#what-it-does)
  - [quickstart](#quickstart)
  - [commands](#commands)
  - [flags & options](#flags--options)
  - [how it works](#how-it-works)
  - [what i learned](#what-i-learned)
  - [stack](#stack)
  - [find me](#find-me)

## what it does

- parses human durations effortlessly (`25m`, `1h30m`, `45s`, `1.5h`, `1d`)
- zero-drift timer engine calculated against absolute timestamps (`TargetEnd = StartedAt + Duration`), resilient to system suspend/sleep
- auto-spawning background daemon communicating through unix domain sockets (or named pipes on windows)
- concurrent multiple timers running independently in the background
- beautiful terminal ui with unicode clock glyphs (`○ ◔ ◑ ◕ ●`), filament gauge (`━━━╸┈┈┈`), and universal osc 2 window/tab titles
- terminal agnostic — clean suppression of ansi sequences when output is piped or redirected
- 4-round pomodoro workflow state machine (work -> short break -> long break)
- native desktop notifications (`notify-send`, `osascript`, windows toast)
- multi-tiered audio cascade (pipewire `pw-play` -> pulseaudio `paplay` -> alsa `aplay` -> embedded 44.1khz wav chime fallback -> terminal bell)
- session history logging and daily analytics stored in pure-go sqlite with wal mode enabled

## quickstart

clone and compile the binary:

```bash
git clone https://github.com/tgr-wjya/timers.git
cd timers
go build -o timer ./cmd/timer
```

put it in your path or run it directly:

```bash
# start a quick 25-minute focus session
./timer 25m

# start a named timer with title
./timer 1h30m -t "deep work"

# mute audio chimes if you're in a library or meeting
./timer 10m -m -t "quick sprint"
```

## commands

| command | what it does | example |
| --- | --- | --- |
| `timer <duration>` | start a new timer with human duration | `timer 25m` or `timer 1h30m` |
| `timer list` | list all active, paused, and recent timers | `timer list` |
| `timer pause <id>` | pause an active timer | `timer pause 1` |
| `timer resume <id>` | resume a paused timer without clock drift | `timer resume 1` |
| `timer cancel <id>` | cancel an active or paused timer | `timer cancel 1` |
| `timer cancel all` | cancel all running timers at once | `timer cancel all` |
| `timer pomodoro [task]` | start an interactive pomodoro cycle | `timer pomodoro "refactoring"` |
| `timer log` | view session history logged to sqlite | `timer log` or `timer log --limit 10` |
| `timer stats` | view focus analytics and completion rate | `timer stats` |
| `timer daemon status` | check background daemon health & pid | `timer daemon status` |
| `timer daemon stop` | stop the running background daemon | `timer daemon stop` |

## flags & options

```bash
# timer duration flags
-t, --title <string>     custom title for the timer (defaults to duration string)
-m, --mute               mute audio alerts (desktop notifications still fire)

# pomodoro cycle customization
--work <duration>        duration of focus block (default: 25m)
--break <duration>       duration of short break (default: 5m)
--long <duration>        duration of long break after 4 rounds (default: 15m)
--rounds <int>           number of focus rounds before long break (default: 4)

# log filtering
-l, --limit <int>        number of historical entries to display (default: 20)
```

## how it works

- **auto-spawning daemon:** when running any cli command, the client pings the local socket. if the daemon isn't running or the socket is stale, it automatically forks a detached background process and waits for socket readiness.
- **zero-drift clock math:** instead of relying on naive `time.Sleep` increments which drift over time or freeze during laptop suspend, the engine tracks absolute wall-clock `TargetEnd = StartedAt + Duration`. when paused, elapsed duration is preserved and `TargetEnd` shifts accordingly upon resume.
- **universal terminal integration:** dynamically detects whether stdout is a tty using `isatty`. when attached to an interactive terminal, it renders live progress with osc 2 terminal title updates. when piped to another tool (`timer list | grep something`), ansi formatting is safely disabled.
- **cgo-free sqlite:** leverages `modernc.org/sqlite` so the binary compiles anywhere without requiring a c toolchain or system libraries, while keeping concurrent wal mode active for fast reads and writes.

## what i learned

- building background daemons in go requires different process isolation per platform. on unix/linux we can use `SysProcAttr{Setsid: true}`, but windows doesn't have `Setsid` and requires `CREATE_NEW_PROCESS_GROUP` / detached flags. keeping them in separate `proc_unix.go` and `proc_windows.go` files keeps builds clean and cross-platform.
- pipewire `pw-play` on nobara/fedora is super fast, but when fallback is needed, embedding an uncompressed 44.1khz wav chime right into the binary with `//go:embed` means audio alerts always work, even in minimal container or vm environments.
- avoiding premature abstractions makes go code so much easier to maintain. a direct package function like `sound.Play()` or `notify.Send()` is way cleaner and less ceremony than building an interface with only one struct implementation.
- xdg directory specifications (`~/.local/share/timer` and `~/.local/state/timer` or `$XDG_RUNTIME_DIR`) keep the filesystem tidy compared to dumping random dotfiles in `$HOME`.

## stack

go + modernc.org/sqlite + unix domain sockets + pipewire / native desktop alerts

## find me

i'm active in boot.dev, you should check me out.

[portfolio website](https://tgr-wjya.up.railway.app/) · [email](mailto:tgrwjya6371+contact@gmail.com) · [linkedin](https://www.linkedin.com/in/tgr-wjya/) · [boot.dev](https://www.boot.dev/u/handmadeinvite39)

---

made with ◉‿◉
