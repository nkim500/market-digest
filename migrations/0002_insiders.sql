CREATE TABLE IF NOT EXISTS insider_trades (
  id             INTEGER PRIMARY KEY,
  source         TEXT NOT NULL,
  filer          TEXT NOT NULL,
  role           TEXT,
  ticker         TEXT,
  asset_desc     TEXT,
  side           TEXT,
  amount_low     INTEGER,
  amount_high    INTEGER,
  transaction_ts INTEGER,
  filing_ts      INTEGER,
  raw_url        TEXT,
  hash           TEXT UNIQUE NOT NULL
);
CREATE INDEX IF NOT EXISTS insider_trades_ticker_ts ON insider_trades(ticker, transaction_ts DESC);
CREATE INDEX IF NOT EXISTS insider_trades_filer ON insider_trades(filer);
CREATE INDEX IF NOT EXISTS insider_trades_filing_ts ON insider_trades(filing_ts DESC);
