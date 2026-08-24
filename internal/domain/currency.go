package domain

import (
	"context"
	"time"
)

// Quote is the central domain entity for currency exchange rates.
// All financial values are scaled (multiplied by 10000) to avoid using float64.
type Quote struct {
	From       string    // Base currency (e.g., "USD")
	To         string    // Target currency (e.g., "EUR")
	Rate       int64     // Spot rate (e.g., 0.9200 -> 9200)
	Volatility int64     // Historical volatility sigma (e.g., 6% -> 0.0600 -> 600)
	ValidUntil time.Time // Expiration time for this price guarantee (Quote TTL)
}

type CurrencyFetcher interface {
	FetchRate(ctx context.Context, from, to string) (Quote, error)
}
