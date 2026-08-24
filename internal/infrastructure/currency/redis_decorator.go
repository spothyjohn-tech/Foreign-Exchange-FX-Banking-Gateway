package currency

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"fx-gateway/internal/domain"

	"github.com/redis/go-redis/v9"
)

type CachedCurrencyFetcher struct{
	rdb *redis.Client
	next domain.CurrencyFetcher
	ttl time.Duration
}

func NewCachedCurrencyFetcher(rdb *redis.Client, next domain.CurrencyFetcher) *CachedCurrencyFetcher{
	return &CachedCurrencyFetcher{
		rdb: rdb,
		next: next,
		ttl: 1*time.Hour, 
	}
}

func (c *CachedCurrencyFetcher) FetchRate(ctx context.Context, from, to string) (domain.Quote,error){
	cacheKey := fmt.Sprintf("ft_rate:%s:%s", from, to)
	cachedJSON, err := c.rdb.Get(ctx, cacheKey).Result()
	if err == nil {
		var quote domain.Quote
		if unserializeErr := json.Unmarshal([]byte(cachedJSON), &quote); unserializeErr == nil {
			return quote, nil 
		}
	}
	quote, err := c.next.FetchRate(ctx, from, to)
	if err != nil {
		return domain.Quote{}, fmt.Errorf("failed to fetch rate from source: %w", err)
	}
	quoteBytes, serializeErr := json.Marshal(quote)
	if serializeErr == nil {
		_ = c.rdb.Set(ctx, cacheKey, string(quoteBytes), c.ttl).Err()
	}

	return quote, nil
}