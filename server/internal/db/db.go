package db

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

// DB wraps the sql.DB connection.
type DB struct {
	*sql.DB
}

// Open opens a SQLite database with WAL mode and performance pragmas.
func Open(dbPath string) (*DB, error) {
	dsn := fmt.Sprintf("%s?_journal_mode=WAL&_synchronous=NORMAL&_busy_timeout=5000&_foreign_keys=ON", dbPath)
	conn, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	conn.SetMaxOpenConns(1) // SQLite single-writer

	db := &DB{conn}
	if err := db.migrate(); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}

	log.Printf("[db] opened %s", dbPath)
	return db, nil
}

// migrate runs all DDL migrations.
func (db *DB) migrate() error {
	migrations := []string{
		// Devices
		`CREATE TABLE IF NOT EXISTS devices (
			device_id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL DEFAULT 'primary',
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			last_seen_at TEXT,
			client_version TEXT,
			status TEXT NOT NULL DEFAULT 'active'
		)`,
		// Auth tokens
		`CREATE TABLE IF NOT EXISTS auth_tokens (
			token_id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL DEFAULT 'primary',
			token_hash TEXT NOT NULL,
			token_type TEXT NOT NULL,
			scopes TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			expires_at TEXT,
			revoked_at TEXT
		)`,
		// Events
		`CREATE TABLE IF NOT EXISTS events (
			event_id TEXT PRIMARY KEY,
			device_id TEXT NOT NULL,
			user_id TEXT NOT NULL DEFAULT 'primary',
			sequence INTEGER NOT NULL,
			timestamp TEXT NOT NULL,
			timezone TEXT NOT NULL,
			event_type TEXT NOT NULL,
			collector TEXT,
			source TEXT,
			schema_version INTEGER NOT NULL DEFAULT 1,
			payload TEXT NOT NULL DEFAULT '{}',
			received_at TEXT NOT NULL DEFAULT (datetime('now'))
		)`,
		`CREATE INDEX IF NOT EXISTS idx_events_device_time ON events(device_id, timestamp DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_events_user_time ON events(user_id, timestamp DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_events_type_time ON events(event_type, timestamp DESC)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_events_device_sequence ON events(device_id, sequence)`,
		// Current context
		`CREATE TABLE IF NOT EXISTS current_context (
			device_id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL DEFAULT 'primary',
			timestamp TEXT NOT NULL,
			timezone TEXT NOT NULL,
			context TEXT NOT NULL DEFAULT '{}',
			etag TEXT NOT NULL DEFAULT '',
			updated_at TEXT NOT NULL DEFAULT (datetime('now'))
		)`,
		// Timeline segments
		`CREATE TABLE IF NOT EXISTS timeline_segments (
			segment_id TEXT PRIMARY KEY,
			device_id TEXT NOT NULL,
			user_id TEXT NOT NULL DEFAULT 'primary',
			segment_type TEXT NOT NULL,
			layer TEXT,
			start_time TEXT NOT NULL,
			end_time TEXT NOT NULL,
			label TEXT NOT NULL,
			confidence REAL DEFAULT 0,
			version INTEGER NOT NULL DEFAULT 1,
			valid_from TEXT NOT NULL,
			valid_to TEXT,
			ingested_at TEXT NOT NULL DEFAULT (datetime('now')),
			revised_at TEXT,
			revision_reason TEXT,
			evidence TEXT,
			payload TEXT NOT NULL DEFAULT '{}',
			received_at TEXT NOT NULL DEFAULT (datetime('now'))
		)`,
		`CREATE INDEX IF NOT EXISTS idx_segments_user_time ON timeline_segments(user_id, start_time DESC, end_time DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_segments_user_type ON timeline_segments(user_id, segment_type, start_time DESC)`,
		// Daily summary
		`CREATE TABLE IF NOT EXISTS daily_summary (
			user_id TEXT NOT NULL DEFAULT 'primary',
			date TEXT NOT NULL,
			timezone TEXT NOT NULL,
			summary TEXT NOT NULL DEFAULT '{}',
			version INTEGER NOT NULL DEFAULT 1,
			updated_at TEXT NOT NULL DEFAULT (datetime('now')),
			PRIMARY KEY (user_id, date)
		)`,
		// Weekly summary
		`CREATE TABLE IF NOT EXISTS weekly_summary (
			user_id TEXT NOT NULL DEFAULT 'primary',
			week TEXT NOT NULL,
			timezone TEXT NOT NULL,
			summary TEXT NOT NULL DEFAULT '{}',
			version INTEGER NOT NULL DEFAULT 1,
			updated_at TEXT NOT NULL DEFAULT (datetime('now')),
			PRIMARY KEY (user_id, week)
		)`,
		// Five-minute summary
		`CREATE TABLE IF NOT EXISTS five_minute_summary (
			user_id TEXT NOT NULL DEFAULT 'primary',
			start_time TEXT NOT NULL,
			end_time TEXT NOT NULL,
			summary TEXT NOT NULL DEFAULT '{}',
			sync_state TEXT NOT NULL DEFAULT 'pending',
			PRIMARY KEY (user_id, start_time)
		)`,
		// Hourly summary
		`CREATE TABLE IF NOT EXISTS hourly_summary (
			user_id TEXT NOT NULL DEFAULT 'primary',
			start_time TEXT NOT NULL,
			end_time TEXT NOT NULL,
			summary TEXT NOT NULL DEFAULT '{}',
			sync_state TEXT NOT NULL DEFAULT 'pending',
			PRIMARY KEY (user_id, start_time)
		)`,
		// Agent observations
		`CREATE TABLE IF NOT EXISTS agent_observations (
			observation_id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL DEFAULT 'primary',
			timestamp TEXT NOT NULL,
			agent_id TEXT,
			observation_type TEXT,
			severity TEXT,
			payload TEXT NOT NULL DEFAULT '{}',
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		)`,
		`CREATE INDEX IF NOT EXISTS idx_agent_obs_time ON agent_observations(user_id, timestamp DESC)`,
		// Interventions
		`CREATE TABLE IF NOT EXISTS interventions (
			intervention_id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL DEFAULT 'primary',
			timestamp TEXT NOT NULL,
			agent_id TEXT,
			intervention_type TEXT,
			status TEXT NOT NULL DEFAULT 'proposed',
			payload TEXT NOT NULL DEFAULT '{}',
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			updated_at TEXT NOT NULL DEFAULT (datetime('now'))
		)`,
		`CREATE INDEX IF NOT EXISTS idx_interventions_time ON interventions(user_id, timestamp DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_interventions_status ON interventions(user_id, status)`,
		// User feedback
		`CREATE TABLE IF NOT EXISTS user_feedback (
			feedback_id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL DEFAULT 'primary',
			timestamp TEXT NOT NULL,
			target_type TEXT NOT NULL,
			target_id TEXT NOT NULL,
			feedback_type TEXT NOT NULL,
			payload TEXT NOT NULL DEFAULT '{}',
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		)`,
		`CREATE INDEX IF NOT EXISTS idx_feedback_target ON user_feedback(user_id, target_type, target_id)`,
		// Goals
		`CREATE TABLE IF NOT EXISTS goals (
			goal_id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL DEFAULT 'primary',
			updated_at TEXT NOT NULL DEFAULT (datetime('now')),
			goal_type TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'active',
			payload TEXT NOT NULL DEFAULT '{}'
		)`,
		// Habits
		`CREATE TABLE IF NOT EXISTS habits (
			habit_id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL DEFAULT 'primary',
			habit_type TEXT NOT NULL,
			target TEXT NOT NULL DEFAULT '{}',
			schedule TEXT NOT NULL DEFAULT '{}',
			evidence_sources TEXT NOT NULL DEFAULT '[]',
			status TEXT NOT NULL DEFAULT 'active',
			streak_days INTEGER NOT NULL DEFAULT 0,
			habit_strength REAL,
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			updated_at TEXT NOT NULL DEFAULT (datetime('now'))
		)`,
		`CREATE INDEX IF NOT EXISTS idx_habits_user ON habits(user_id)`,
		// Baselines
		`CREATE TABLE IF NOT EXISTS baselines (
			baseline_id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL DEFAULT 'primary',
			metric_name TEXT NOT NULL,
			window_days INTEGER NOT NULL DEFAULT 7,
			value REAL NOT NULL,
			computed_at TEXT NOT NULL DEFAULT (datetime('now')),
			updated_at TEXT NOT NULL DEFAULT (datetime('now')),
			UNIQUE(user_id, metric_name)
		)`,
		// Outcomes
		`CREATE TABLE IF NOT EXISTS outcomes (
			outcome_id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL DEFAULT 'primary',
			intervention_id TEXT NOT NULL,
			timestamp TEXT NOT NULL,
			outcome_type TEXT NOT NULL,
			effectiveness_score REAL,
			payload TEXT NOT NULL DEFAULT '{}',
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		)`,
		`CREATE INDEX IF NOT EXISTS idx_outcomes_intervention ON outcomes(intervention_id)`,
		`CREATE INDEX IF NOT EXISTS idx_outcomes_user_time ON outcomes(user_id, timestamp DESC)`,
		// Audit log
		`CREATE TABLE IF NOT EXISTS audit_log (
			audit_id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL DEFAULT 'primary',
			actor_type TEXT NOT NULL,
			actor_id TEXT,
			action TEXT NOT NULL,
			target_type TEXT,
			target_id TEXT,
			metadata TEXT,
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_user_time ON audit_log(user_id, created_at DESC)`,
		// Policy settings
		`CREATE TABLE IF NOT EXISTS policy_settings (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			settings TEXT NOT NULL DEFAULT '{}',
			updated_at TEXT NOT NULL DEFAULT (datetime('now'))
		)`,
	}

	for _, m := range migrations {
		if _, err := db.Exec(m); err != nil {
			return fmt.Errorf("migration failed: %s: %w", m[:60], err)
		}
	}

	log.Println("[db] migrations complete")
	return nil
}

// Close closes the database.
func (db *DB) Close() error {
	return db.DB.Close()
}
