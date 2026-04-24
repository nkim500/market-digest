// Package insiders holds the types and fetchers for politician/corporate
// insider trade data.
package insiders

// Trade is the normalized row written to insider_trades.
type Trade struct {
	Source        string // 'senate' | 'house' | 'sec-form4'
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
}
