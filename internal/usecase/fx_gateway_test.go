package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"fx-gateway/internal/domain"
	"fx-gateway/internal/infrastructure/repository"
)

type stubFetcher struct {
	rateToReturn int64
	volToReturn  int64
	errToReturn error
}

func (f stubFetcher) FetchRate(ctx context.Context, from, to string) (domain.Quote, error) {
	if f.errToReturn != nil {
		return domain.Quote{}, f.errToReturn
	}
	
	return domain.Quote{
		From:       from,
		To:         to,
		Rate:       f.rateToReturn,
		Volatility: f.volToReturn, 
		ValidUntil: time.Now().Add(5 * time.Minute),
	}, nil
}
type stubLogger struct{
	errToReturn error
}

func (l stubLogger) LogTransaction(ctx context.Context, tx repository.TransactionEntity) error{
	return l.errToReturn
}

func TestFXGatewayUseCase_Execute(t *testing.T){
	type testCase struct {
		name          string // Имя сценария
		inputAmount   int64  // Сколько центов передает SaaS
		mockRate      int64  // Какой курс «вернет» наш стаб
		mockVol      int64 // Новое поле для симуляции волатильности пары
		mockFetchErr  error  // Какую ошибку сети «выдаст» наш стаб
		mockLogErr    error  // Какую ошибку БД «выдаст» наш стаб
		wantAmount    int64  // Сколько центов мы ОЖИДАЕМ получить на выходе
		wantErr       bool   // Ожидаем ли мы вообще ошибку от Use Case
	}

	// Заполняем таблицу сценариями
	table := []testCase{
		{
			name:        "Успешный перевод USD в EUR с динамическим спредом (EUR Vol = 6%)",
			inputAmount: 10000, // 100.00 USD
			mockRate:    9200,  // Курс 0.9200
			mockVol:     600,
			// Математический эталон Quants-движка:
			// volatilitySpreadBps = (600 * 1217) / 10000 = 73 bps
			// totalSpreadBps = 73 + 50 = 123 bps (1.23%)
			// rawTargetAmount = (10000 * 9200) / 10000 = 9200
			// targetAmountWithValues = (9200 * 10123) / 10000 = 9313
			wantAmount:  9313,  
			wantErr:     false,
		},
		{
			name:         "Падение внешнего API провайдера валют",
			inputAmount:  10000,
			mockRate:     0,
			mockVol:      0,
			mockFetchErr: errors.New("external provider timeout"), // имитируем падение сети
			wantAmount:   0,
			wantErr:      true, // Use Case должен вернуть ошибку наверх
		},
		{
			name:        "База данных биллинга упала",
			inputAmount: 10000,
			mockRate:    9200,
			mockVol:     600,
			mockLogErr:  errors.New("postgres connection refused"), // имитируем падение БД
			wantAmount:  0,
			wantErr:     true, // Финтех-правило: если лог не записан, Use Case должен вернуть ошибку!
		},
	}

	for _, tc := range table{
		t.Run(tc.name, func(t *testing.T){
			fetcher := stubFetcher{rateToReturn: tc.mockRate, volToReturn: tc.mockVol, errToReturn: tc.mockFetchErr}
			logger := stubLogger{errToReturn: tc.mockLogErr}
			uc := NewFXGatewayUseCase(fetcher, logger, 0)
			
			req := ConvertAndBillingRequest{
				ClientID: "test_client",
				BaseAmount: tc.inputAmount,
				BaseCurrency: "USD",
				TargetCurrency: "EUR",
			}
			resp, err := uc.Execute(context.Background(), req)
			if (err != nil) != tc.wantErr {
				t.Fatalf("Ожидался флаг ошибки %v, но получено err = %v", tc.wantErr, err)
			}
			if !tc.wantErr {
				if resp.TargetAmount != tc.wantAmount {
					t.Errorf("ОШИБКА РАСЧЕТА СПРЕДА: Хотели получить %d центов, но Use Case насчитал %d", tc.wantAmount, resp.TargetAmount)
				}
			}
		})
	}
}