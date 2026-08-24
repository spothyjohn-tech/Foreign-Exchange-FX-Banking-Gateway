package usecase

import(
	"context"
	"fmt"
	"log/slog"
	"time"

	"fx-gateway/internal/domain"
	"fx-gateway/internal/infrastructure/repository"
)

type TxLogger interface {
	LogTransaction(ctx context.Context, tx repository.TransactionEntity) error
}

type FXGatewayUseCase struct {
	fetcher domain.CurrencyFetcher
	txLogger TxLogger
	spread int32
}

func NewFXGatewayUseCase(fetcher domain.CurrencyFetcher, txLogger TxLogger, spreadBps int32) *FXGatewayUseCase {
	return &FXGatewayUseCase{
		fetcher: fetcher,
		txLogger: txLogger,
		spread: spreadBps,
	}
}

type ConvertAndBillingRequest struct {
	ClientID string
	BaseAmount int64
	BaseCurrency string
	TargetCurrency string

}

type ConvertAndBillingResponse struct {
	TargetAmount int64
	AppliedRate int64
}

func (uc *FXGatewayUseCase) Execute(ctx context.Context, req ConvertAndBillingRequest) (*ConvertAndBillingResponse, error){
	if req.ClientID == "" {
		return nil, domain.ErrMissingClientID
	}
	if req.BaseAmount <= 0 {
		return nil, domain.ErrInvalidAmount
	}
	if len(req.BaseCurrency) != 3 || len(req.TargetCurrency) != 3 {
		return nil, domain.ErrInvalidCurrency
	}
	
	
	quote, err := uc.fetcher.FetchRate(ctx, req.BaseCurrency, req.TargetCurrency)
	if err != nil {
		return nil, fmt.Errorf("usecase: failed to get rate: %w", err)
	}

	volatilitySpreadBps := (quote.Volatility * 1217) / 10000
	totalSpreadBps := volatilitySpreadBps + 50

	rawTargetAmount := (req.BaseAmount * quote.Rate) / 10000
	targetAmountWithValues := (rawTargetAmount * (10000 + int64(totalSpreadBps))) / 10000
	
	dbTx := repository.TransactionEntity{
		ClientID: req.ClientID,
		BaseCurrency: req.BaseCurrency,
		TargetCurrency: req.TargetCurrency,
		BaseAmount:     req.BaseAmount,
		TargetAmount:   targetAmountWithValues,
		AppliedRate:    quote.Rate,
		SpreadBps:      uc.spread,
		CreatedAt:      time.Now(),
	}
	if err := uc.txLogger.LogTransaction(ctx, dbTx); err != nil{
		return nil, fmt.Errorf("usecase: billing log failed: %w", err)
	}
	return &ConvertAndBillingResponse{
		TargetAmount: targetAmountWithValues,
		AppliedRate: quote.Rate,
	}, nil
}

func (uc *FXGatewayUseCase) FetchAndSaveRates(ctx context.Context) error {
	targetCurrencies := []string{"EUR", "USD", "JPY"}
	baseCurrency := "USD"

	for _, target := range targetCurrencies{
		_, err := uc.fetcher.FetchRate(ctx,baseCurrency,target)
		if err != nil {
				slog.Error("worker_usecase: failed to background update %s/%s: %v\n", baseCurrency, target, "error", err)
				continue
		}
	}
	return nil
}