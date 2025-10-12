package db

import (
	"database/sql"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Store struct{ *sql.DB }

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	// --- Ajustes recomendados para concurrencia y rendimiento ---
	// WAL permite lecturas concurrentes mientras se escribe.
	_, _ = db.Exec(`PRAGMA journal_mode=WAL;`)
	// Menos fsyncs, suficiente para logs/métricas.
	_, _ = db.Exec(`PRAGMA synchronous=NORMAL;`)
	// Esperar hasta 5s si la DB está ocupada (evita "database is locked").
	_, _ = db.Exec(`PRAGMA busy_timeout=5000;`)

	if err := migrate(db); err != nil {
		return nil, err
	}
	return &Store{db}, nil
}

func migrate(db *sql.DB) error {
	_, err := db.Exec(`
CREATE TABLE IF NOT EXISTS system_metrics (
  ts INTEGER NOT NULL,
  pid INTEGER, name TEXT, state TEXT, cmdline TEXT,
  vsz_kb INTEGER, rss_kb INTEGER, mem_pct REAL, cpu_pct REAL
);
CREATE TABLE IF NOT EXISTS container_metrics (
  ts INTEGER NOT NULL,
  container_id TEXT, shim_pid INTEGER, shim_name TEXT,
  pid INTEGER, name TEXT, cmdline TEXT,
  vsz_kb INTEGER, rss_kb INTEGER, mem_pct REAL, cpu_pct REAL
);
CREATE TABLE IF NOT EXISTS actions_log (
  ts INTEGER NOT NULL,
  action TEXT, container_id TEXT, reason TEXT, details TEXT
);
CREATE TABLE IF NOT EXISTS host_metrics (
  ts INTEGER NOT NULL,
  totalram_kb INTEGER,
  freeram_kb INTEGER
);
CREATE INDEX IF NOT EXISTS idx_host_ts ON host_metrics(ts);
-- Índices útiles
CREATE INDEX IF NOT EXISTS idx_sys_ts ON system_metrics(ts);
CREATE INDEX IF NOT EXISTS idx_cont_ts ON container_metrics(ts);
CREATE INDEX IF NOT EXISTS idx_cont_cid ON container_metrics(container_id);
`)
	return err
}

func nowTS() int64 { return time.Now().Unix() }

// ===== Insert helpers "auto-ts" (las tuyas originales) =====

func (s *Store) InsertSystemProc(pid int, name, state, cmd string, vsz, rss uint64, mem, cpu float64) error {
	_, err := s.Exec(
		`INSERT INTO system_metrics (ts,pid,name,state,cmdline,vsz_kb,rss_kb,mem_pct,cpu_pct)
		 VALUES (?,?,?,?,?,?,?,?,?)`,
		nowTS(), pid, name, state, cmd, vsz, rss, mem, cpu,
	)
	return err
}

func (s *Store) InsertContainerProc(cid string, shimPID int, shimName string, pid int, name, cmd string, vsz, rss uint64, mem, cpu float64) error {
	_, err := s.Exec(
		`INSERT INTO container_metrics (ts,container_id,shim_pid,shim_name,pid,name,cmdline,vsz_kb,rss_kb,mem_pct,cpu_pct)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		nowTS(), cid, shimPID, shimName, pid, name, cmd, vsz, rss, mem, cpu,
	)
	return err
}

func (s *Store) LogAction(action, cid, reason, details string) error {
	_, err := s.Exec(
		`INSERT INTO actions_log (ts,action,container_id,reason,details) VALUES (?,?,?,?,?)`,
		nowTS(), action, cid, reason, details,
	)
	return err
}

// ===== Opcionales: variantes con ts explícito (por si las quieres usar) =====

func (s *Store) InsertSystemProcWithTS(ts int64, pid int, name, state, cmd string, vsz, rss uint64, mem, cpu float64) error {
	_, err := s.Exec(
		`INSERT INTO system_metrics (ts,pid,name,state,cmdline,vsz_kb,rss_kb,mem_pct,cpu_pct)
		 VALUES (?,?,?,?,?,?,?,?,?)`,
		ts, pid, name, state, cmd, vsz, rss, mem, cpu,
	)
	return err
}

func (s *Store) InsertContainerProcWithTS(ts int64, cid string, shimPID int, shimName string, pid int, name, cmd string, vsz, rss uint64, mem, cpu float64) error {
	_, err := s.Exec(
		`INSERT INTO container_metrics (ts,container_id,shim_pid,shim_name,pid,name,cmdline,vsz_kb,rss_kb,mem_pct,cpu_pct)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		ts, cid, shimPID, shimName, pid, name, cmd, vsz, rss, mem, cpu,
	)
	return err
}
