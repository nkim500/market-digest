// Package insiders holds the types and fetchers for politician/corporate
// insider trade data.
package insiders

// Trade is the normalized row written to insider_trades.
type Trade struct {
	Source        string // 'senate' | 'house' | 'sec-form4' | 'finnhub' | 'quiver'
	Filer         string
	Role          string
	Ticker        string
	AssetDesc     string
	Side          string // 'buy' | 'sell' | 'exchange'
	AmountLow     int    // USD
	AmountHigh    int
	TransactionTS int64 // unix
	FilingTS      int64
	RawURL        string
	Hash          string

	// Form 4-specific fields. Politicians leave these zero/empty; the store
	// layer writes them as SQL NULL when unset so the partial index on
	// transaction_code stays clean.
	Shares          *int     // share count (signed: +acquired / -disposed)
	PricePerShare   *float64 // execution price USD
	TransactionCode string   // single-char SEC code: P S A M F G D ...
	SecurityType    string   // 'common' | 'derivative'
}
