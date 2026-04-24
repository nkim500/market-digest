ALTER TABLE insider_trades ADD COLUMN shares INTEGER;
ALTER TABLE insider_trades ADD COLUMN price_per_share REAL;
ALTER TABLE insider_trades ADD COLUMN transaction_code TEXT;
ALTER TABLE insider_trades ADD COLUMN security_type TEXT;

CREATE INDEX IF NOT EXISTS insider_trades_txcode
  ON insider_trades(transaction_code) WHERE transaction_code IS NOT NULL;
