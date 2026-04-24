-- 0001_init.sql
CREATE TABLE IF NOT EXISTS watchlist (
  ticker      TEXT PRIMARY KEY,
  note        TEXT,
  added_ts    INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS alerts (
  id          INTEGER PRIMARY KEY,
  created_ts  INTEGER NOT NULL,
  source      TEXT NOT NULL,
  severity    TEXT NOT NULL,
  ticker      TEXT,
  title       TEXT NOT NULL,
  body        TEXT,
  payload     TEXT,
  seen_ts     INTEGER
);
CREATE INDEX IF NOT EXISTS alerts_unseen ON alerts(seen_ts) WHERE seen_ts IS NULL;
CREATE INDEX IF NOT EXISTS alerts_created ON alerts(created_ts DESC);

CREATE TABLE IF NOT EXISTS job_runs (
  id          INTEGER PRIMARY KEY,
  job         TEXT NOT NULL,
  started_ts  INTEGER NOT NULL,
  finished_ts INTEGER,
  status      TEXT NOT NULL,
  rows_in     INTEGER,
  rows_new    INTEGER,
  error       TEXT
);
CREATE INDEX IF NOT EXISTS job_runs_job_started ON job_runs(job, started_ts DESC);

CREATE TABLE IF NOT EXISTS schema_version (
  version     INTEGER PRIMARY KEY,
  applied_ts  INTEGER NOT NULL
);
