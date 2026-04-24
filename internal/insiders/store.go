package insiders

import (
	"context"
	"database/sql"
	"fmt"
)

// StoreInserts upserts trades with INSERT OR IGNORE on hash. Returns the row
// ids of actually-inserted rows so the caller can evaluate alert rules only
// against new data.
func StoreInserts(ctx context.Context, conn *sql.DB, trades []Trade) ([]int64, error) {
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT OR IGNORE INTO insider_trades
		  (source, filer, role, ticker, asset_desc, side,
		   amount_low, amount_high, transaction_ts, filing_ts, raw_url, hash)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	var ids []int64
	for _, t := range trades {
		res, err := stmt.ExecContext(ctx,
			t.Source, t.Filer, t.Role, nullStr(t.Ticker), t.AssetDesc, t.Side,
			t.AmountLow, t.AmountHigh, t.TransactionTS, t.FilingTS, t.RawURL, t.Hash,
		)
		if err != nil {
			return nil, fmt.Errorf("insert trade hash=%s: %w", t.Hash, err)
		}
		affected, err := res.RowsAffected()
		if err != nil {
			return nil, err
		}
		if affected == 1 {
			id, err := res.LastInsertId()
			if err != nil {
				return nil, err
			}
			ids = append(ids, id)
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return ids, nil
}

func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}
