package insiders

import (
	"context"
	"database/sql"
	"fmt"
)

// StoreResult holds the outcome of a StoreInserts call.
type StoreResult struct {
	IDs    []int64 // row IDs of actually-inserted trades
	Trades []Trade // the Trade values that were actually inserted (parallel to IDs)
}

// StoreInserts upserts trades with INSERT OR IGNORE on hash. Returns a
// StoreResult with the row IDs and Trade values of actually-inserted rows so
// the caller can evaluate alert rules only against new data without an
// additional per-row SELECT.
func StoreInserts(ctx context.Context, conn *sql.DB, trades []Trade) (StoreResult, error) {
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return StoreResult{}, err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT OR IGNORE INTO insider_trades
		  (source, filer, role, ticker, asset_desc, side,
		   amount_low, amount_high, transaction_ts, filing_ts, raw_url, hash,
		   shares, price_per_share, transaction_code, security_type)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return StoreResult{}, err
	}
	defer stmt.Close()

	var result StoreResult
	for _, t := range trades {
		res, err := stmt.ExecContext(ctx,
			t.Source, t.Filer, t.Role, nullStr(t.Ticker), t.AssetDesc, t.Side,
			t.AmountLow, t.AmountHigh, t.TransactionTS, t.FilingTS, t.RawURL, t.Hash,
			nullIntPtr(t.Shares), nullFloatPtr(t.PricePerShare),
			nullStr(t.TransactionCode), nullStr(t.SecurityType),
		)
		if err != nil {
			return StoreResult{}, fmt.Errorf("insert trade hash=%s: %w", t.Hash, err)
		}
		affected, err := res.RowsAffected()
		if err != nil {
			return StoreResult{}, err
		}
		if affected == 1 {
			id, err := res.LastInsertId()
			if err != nil {
				return StoreResult{}, err
			}
			result.IDs = append(result.IDs, id)
			result.Trades = append(result.Trades, t)
		}
	}
	if err := tx.Commit(); err != nil {
		return StoreResult{}, err
	}
	return result, nil
}

func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func nullIntPtr(p *int) any {
	if p == nil {
		return nil
	}
	return *p
}

func nullFloatPtr(p *float64) any {
	if p == nil {
		return nil
	}
	return *p
}
