package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"timers/internal/pomodoro"
	"timers/internal/timer"

	_ "modernc.org/sqlite"
)

type Stats struct {
	TotalSessions     int
	CompletedSessions int
	TotalFocusSec     int
	TopTasks          []TaskStat
}

type TaskStat struct {
	Title    string
	Duration time.Duration
	Count    int
}

type Store struct {
	db *sql.DB
}

func NewStore(dbPath string) (*Store, error) {
	if dbPath != ":memory:" {
		dir := filepath.Dir(dbPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create db directory %q: %w", dir, err)
		}
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database %q: %w", dbPath, err)
	}

	// In-memory sqlite should not enable WAL
	if dbPath != ":memory:" {
		if _, err := db.Exec("PRAGMA journal_mode = WAL;"); err != nil {
			// ignore on non-file systems
		}
	}
	if _, err := db.Exec("PRAGMA busy_timeout = 5000;"); err != nil {
		return nil, err
	}

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migration failed: %w", err)
	}

	return s, nil
}

func (s *Store) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS timers (
		id INTEGER PRIMARY KEY,
		title TEXT NOT NULL,
		category TEXT NOT NULL DEFAULT 'general',
		duration_sec INTEGER NOT NULL,
		started_at TEXT NOT NULL,
		target_end TEXT NOT NULL,
		ended_at TEXT NULL,
		status TEXT NOT NULL,
		remaining_sec INTEGER NOT NULL DEFAULT 0,
		pomodoro_cycle_id TEXT NULL,
		tags TEXT NOT NULL DEFAULT ''
	);

	CREATE TABLE IF NOT EXISTS pomodoro_cycles (
		id TEXT PRIMARY KEY,
		task_name TEXT NOT NULL,
		current_round INTEGER NOT NULL DEFAULT 1,
		total_rounds INTEGER NOT NULL DEFAULT 4,
		state TEXT NOT NULL,
		started_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	);
	`
	_, err := s.db.Exec(schema)
	return err
}

func (s *Store) SaveTimer(t *timer.Timer) error {
	query := `
	INSERT INTO timers (id, title, category, duration_sec, started_at, target_end, ended_at, status, remaining_sec, tags)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET
		title=excluded.title,
		category=excluded.category,
		duration_sec=excluded.duration_sec,
		started_at=excluded.started_at,
		target_end=excluded.target_end,
		ended_at=excluded.ended_at,
		status=excluded.status,
		remaining_sec=excluded.remaining_sec,
		tags=excluded.tags;
	`
	var endedStr *string
	if t.EndedAt != nil {
		str := t.EndedAt.Format(time.RFC3339Nano)
		endedStr = &str
	}

	tagsJoined := strings.Join(t.Tags, ",")
	remSec := int(t.RemainingWhenPaused.Seconds())

	_, err := s.db.Exec(query,
		t.ID,
		t.Title,
		string(t.Category),
		int(t.Duration.Seconds()),
		t.StartedAt.Format(time.RFC3339Nano),
		t.TargetEnd.Format(time.RFC3339Nano),
		endedStr,
		string(t.Status),
		remSec,
		tagsJoined,
	)
	return err
}

func (s *Store) UpdateTimer(t *timer.Timer) error {
	return s.SaveTimer(t)
}

func (s *Store) GetActiveTimers() ([]*timer.Timer, error) {
	query := `
	SELECT id, title, category, duration_sec, started_at, target_end, ended_at, status, remaining_sec, tags
	FROM timers
	WHERE status IN ('running', 'paused')
	ORDER BY id ASC;
	`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanTimers(rows)
}

func (s *Store) GetHistory(since time.Time, limit int) ([]*timer.Timer, error) {
	query := `
	SELECT id, title, category, duration_sec, started_at, target_end, ended_at, status, remaining_sec, tags
	FROM timers
	WHERE started_at >= ?
	ORDER BY started_at DESC
	`
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := s.db.Query(query, since.Format(time.RFC3339Nano))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanTimers(rows)
}

func (s *Store) GetStats(since time.Time) (Stats, error) {
	var stats Stats
	query := `
	SELECT 
		COUNT(*),
		SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END),
		SUM(CASE WHEN status = 'completed' THEN duration_sec ELSE 0 END)
	FROM timers
	WHERE started_at >= ?;
	`
	var total, completed, focusSec sql.NullInt64
	err := s.db.QueryRow(query, since.Format(time.RFC3339Nano)).Scan(&total, &completed, &focusSec)
	if err != nil {
		return stats, err
	}

	if total.Valid {
		stats.TotalSessions = int(total.Int64)
	}
	if completed.Valid {
		stats.CompletedSessions = int(completed.Int64)
	}
	if focusSec.Valid {
		stats.TotalFocusSec = int(focusSec.Int64)
	}

	// Query top tasks
	topQuery := `
	SELECT title, SUM(duration_sec), COUNT(*)
	FROM timers
	WHERE started_at >= ? AND status = 'completed'
	GROUP BY title
	ORDER BY SUM(duration_sec) DESC
	LIMIT 5;
	`
	rows, err := s.db.Query(topQuery, since.Format(time.RFC3339Nano))
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var ts TaskStat
			var durSec int
			if err := rows.Scan(&ts.Title, &durSec, &ts.Count); err == nil {
				ts.Duration = time.Duration(durSec) * time.Second
				stats.TopTasks = append(stats.TopTasks, ts)
			}
		}
	}

	return stats, nil
}

func (s *Store) SavePomodoroCycle(c *pomodoro.Cycle) error {
	query := `
	INSERT INTO pomodoro_cycles (id, task_name, current_round, total_rounds, state, started_at, updated_at)
	VALUES (?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET
		task_name=excluded.task_name,
		current_round=excluded.current_round,
		total_rounds=excluded.total_rounds,
		state=excluded.state,
		updated_at=excluded.updated_at;
	`
	_, err := s.db.Exec(query,
		c.ID,
		c.TaskName,
		c.Round,
		c.Config.TotalRounds,
		string(c.CurrentStage),
		c.StartedAt.Format(time.RFC3339Nano),
		c.UpdatedAt.Format(time.RFC3339Nano),
	)
	return err
}

func (s *Store) Close() error {
	return s.db.Close()
}

func scanTimers(rows *sql.Rows) ([]*timer.Timer, error) {
	var list []*timer.Timer
	for rows.Next() {
		var (
			id          int
			title       string
			category    string
			durationSec int
			startedStr  string
			targetStr   string
			endedStr    sql.NullString
			status      string
			remSec      int
			tagsStr     string
		)

		err := rows.Scan(&id, &title, &category, &durationSec, &startedStr, &targetStr, &endedStr, &status, &remSec, &tagsStr)
		if err != nil {
			return nil, err
		}

		startedAt, _ := time.Parse(time.RFC3339Nano, startedStr)
		targetEnd, _ := time.Parse(time.RFC3339Nano, targetStr)
		var endedAt *time.Time
		if endedStr.Valid {
			t, err := time.Parse(time.RFC3339Nano, endedStr.String)
			if err == nil {
				endedAt = &t
			}
		}

		var tags []string
		if tagsStr != "" {
			tags = strings.Split(tagsStr, ",")
		}

		list = append(list, &timer.Timer{
			ID:                  id,
			Title:               title,
			Category:            timer.Category(category),
			Duration:            time.Duration(durationSec) * time.Second,
			StartedAt:           startedAt,
			TargetEnd:           targetEnd,
			EndedAt:             endedAt,
			Status:              timer.Status(status),
			RemainingWhenPaused: time.Duration(remSec) * time.Second,
			Tags:                tags,
		})
	}
	return list, nil
}
