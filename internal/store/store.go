package store

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"

	"github.com/yusufjaelani/pulseops/internal/model"
)

type Store struct {
	db *sql.DB
}

func New(dbPath string) (*Store, error) {
	if dbPath != ":memory:" {
		dir := filepath.Dir(dbPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed creating db dir: %w", err)
		}
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed opening sqlite: %w", err)
	}

	// SQLite connection settings for concurrency
	db.SetMaxOpenConns(1)

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed migrating schema: %w", err)
	}

	return s, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) migrate() error {
	query := `
	PRAGMA foreign_keys = ON;
	CREATE TABLE IF NOT EXISTS targets (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		url TEXT NOT NULL UNIQUE,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS probe_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		target_id INTEGER NOT NULL,
		status_code INTEGER NOT NULL,
		is_up BOOLEAN NOT NULL,
		latency_ms INTEGER NOT NULL,
		ssl_expiry_days INTEGER NOT NULL,
		error_msg TEXT,
		headers TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(target_id) REFERENCES targets(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_probe_logs_target_created 
	ON probe_logs(target_id, created_at DESC);
	`
	_, err := s.db.Exec(query)
	return err
}

func (s *Store) AddTarget(name, url string) (*model.Target, error) {
	query := `INSERT INTO targets (name, url, created_at) VALUES (?, ?, ?) RETURNING id, name, url, created_at`
	now := time.Now().UTC()
	var t model.Target
	err := s.db.QueryRow(query, name, url, now).Scan(&t.ID, &t.Name, &t.URL, &t.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (s *Store) GetTarget(id int64) (*model.Target, error) {
	query := `SELECT id, name, url, created_at FROM targets WHERE id = ?`
	var t model.Target
	err := s.db.QueryRow(query, id).Scan(&t.ID, &t.Name, &t.URL, &t.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (s *Store) GetTargetByURL(url string) (*model.Target, error) {
	query := `SELECT id, name, url, created_at FROM targets WHERE url = ?`
	var t model.Target
	err := s.db.QueryRow(query, url).Scan(&t.ID, &t.Name, &t.URL, &t.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (s *Store) ListTargets() ([]model.Target, error) {
	query := `SELECT id, name, url, created_at FROM targets ORDER BY id ASC`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var targets []model.Target
	for rows.Next() {
		var t model.Target
		if err := rows.Scan(&t.ID, &t.Name, &t.URL, &t.CreatedAt); err != nil {
			return nil, err
		}
		targets = append(targets, t)
	}
	return targets, rows.Err()
}

func (s *Store) DeleteTarget(id int64) error {
	query := `DELETE FROM targets WHERE id = ?`
	res, err := s.db.Exec(query, id)
	if err != nil {
		return err
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return errors.New("target not found")
	}
	return nil
}

func (s *Store) SaveProbeLog(log *model.ProbeLog) error {
	query := `
	INSERT INTO probe_logs (target_id, status_code, is_up, latency_ms, ssl_expiry_days, error_msg, headers, created_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	RETURNING id, created_at`
	now := time.Now().UTC()
	return s.db.QueryRow(
		query,
		log.TargetID,
		log.StatusCode,
		log.IsUp,
		log.LatencyMs,
		log.SSLExpiryDays,
		log.ErrorMsg,
		log.Headers,
		now,
	).Scan(&log.ID, &log.CreatedAt)
}

func (s *Store) GetLastProbe(targetID int64) (*model.ProbeLog, error) {
	query := `
	SELECT id, target_id, status_code, is_up, latency_ms, ssl_expiry_days, error_msg, headers, created_at
	FROM probe_logs
	WHERE target_id = ?
	ORDER BY created_at DESC
	LIMIT 1`

	var p model.ProbeLog
	var errStr, headersStr sql.NullString
	err := s.db.QueryRow(query, targetID).Scan(
		&p.ID,
		&p.TargetID,
		&p.StatusCode,
		&p.IsUp,
		&p.LatencyMs,
		&p.SSLExpiryDays,
		&errStr,
		&headersStr,
		&p.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if errStr.Valid {
		p.ErrorMsg = errStr.String
	}
	if headersStr.Valid {
		p.Headers = headersStr.String
	}
	return &p, nil
}

func (s *Store) GetProbeHistory(targetID int64, limit int) ([]model.ProbeLog, error) {
	if limit <= 0 {
		limit = 50
	}
	query := `
	SELECT id, target_id, status_code, is_up, latency_ms, ssl_expiry_days, error_msg, headers, created_at
	FROM probe_logs
	WHERE target_id = ?
	ORDER BY created_at DESC
	LIMIT ?`

	rows, err := s.db.Query(query, targetID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []model.ProbeLog
	for rows.Next() {
		var p model.ProbeLog
		var errStr, headersStr sql.NullString
		if err := rows.Scan(
			&p.ID,
			&p.TargetID,
			&p.StatusCode,
			&p.IsUp,
			&p.LatencyMs,
			&p.SSLExpiryDays,
			&errStr,
			&headersStr,
			&p.CreatedAt,
		); err != nil {
			return nil, err
		}
		if errStr.Valid {
			p.ErrorMsg = errStr.String
		}
		if headersStr.Valid {
			p.Headers = headersStr.String
		}
		logs = append(logs, p)
	}
	return logs, rows.Err()
}

func (s *Store) GetTargetSummaries() ([]model.TargetSummary, error) {
	targets, err := s.ListTargets()
	if err != nil {
		return nil, err
	}

	summaries := make([]model.TargetSummary, 0, len(targets))
	for _, t := range targets {
		last, err := s.GetLastProbe(t.ID)
		if err != nil {
			return nil, err
		}

		// Calculate stats from last 50 probes
		history, err := s.GetProbeHistory(t.ID, 50)
		if err != nil {
			return nil, err
		}

		var uptimePct float64 = 100.0
		var avgLatency float64 = 0
		total := len(history)
		if total > 0 {
			upCount := 0
			var totalLat int64
			for _, h := range history {
				if h.IsUp {
					upCount++
				}
				totalLat += h.LatencyMs
			}
			uptimePct = (float64(upCount) / float64(total)) * 100.0
			avgLatency = float64(totalLat) / float64(total)
		}

		summaries = append(summaries, model.TargetSummary{
			Target:        t,
			LastProbe:     last,
			UptimePercent: uptimePct,
			AvgLatencyMs:  avgLatency,
			TotalProbes:   total,
		})
	}

	return summaries, nil
}
