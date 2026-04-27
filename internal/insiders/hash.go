package insiders

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"strings"
)

// Hash is the dedup key. Inputs are normalized (trim, uppercase ticker,
// collapse internal whitespace in filer) so cosmetic differences in upstream
// data don't produce duplicates.
func Hash(t Trade) string {
	filer := strings.Join(strings.Fields(t.Filer), " ")
	h := sha1.Sum([]byte(fmt.Sprintf(
		"%s|%s|%s|%d|%d|%d|%s",
		t.Source,
		filer,
		strings.ToUpper(t.Ticker),
		t.TransactionTS,
		t.AmountLow,
		t.AmountHigh,
		strings.ToLower(t.Side),
	)))
	return hex.EncodeToString(h[:])
}

// HashForm4 is the dedup key for SEC Form 4 trades. Each Form 4 filing has
// a unique accession number, and each filing contains 1..N indexed
// transactions. (accession, txnIndex) is SEC's guaranteed-unique identifier.
func HashForm4(accession string, txnIndex int) string {
	h := sha1.Sum([]byte(fmt.Sprintf("sec-form4|%s|%d", accession, txnIndex)))
	return hex.EncodeToString(h[:])
}
