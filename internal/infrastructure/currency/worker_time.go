package currency

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

type CurrencyUseCase interface{
	FetchAndSaveRates(ctx context.Context) error
}

type CurrencyWorker struct{
	uc CurrencyUseCase
	interval time.Duration
}

func NewCurrencyWorker(uc CurrencyUseCase, interval time.Duration) *CurrencyWorker{
	return &CurrencyWorker{
		uc: uc,
		interval: interval,
	}	
}

func (w *CurrencyWorker) Start(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()

		ticker := time.NewTicker(w.interval)
		defer ticker.Stop()

		slog.Info("Worker valutes succefully start", "interval", w.interval.String())
		go func(){
			warmupCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()
			if err := w.uc.FetchAndSaveRates(warmupCtx); err != nil{
				slog.Info("Worker warmup failed", "error", err)
			}
		}()
		for {
			select{
			case <- ticker.C:
				slog.Info("Timer worked: start update valutes")
				loopCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
				defer cancel()
				if err := w.uc.FetchAndSaveRates(loopCtx); err != nil {
					slog.Error("Worker iteration failed", "error", err)
				}
			case <-ctx.Done():
				slog.Info("Worker received stop signal, shutting down gracefully...")
				return 
			}	
		}
	}()
}