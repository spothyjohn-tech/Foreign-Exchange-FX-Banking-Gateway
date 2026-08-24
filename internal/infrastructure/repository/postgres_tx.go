package repository

import(
	"context"
	"database/sql"
	"fmt"
	"time"
)

type TransactionEntity struct{
	ClientID string
	BaseCurrency string
	TargetCurrency string
	BaseAmount int64
	TargetAmount int64
	AppliedRate int64
	SpreadBps int32
	CreatedAt time.Time
}

type PostgresTxRepository struct{
	db *sql.DB
}

func NewPostgresTxRepository(db *sql.DB) *PostgresTxRepository {
	return &PostgresTxRepository{db:db}
}

func (r *PostgresTxRepository) LogTransaction(ctx context.Context, tx TransactionEntity) error{
	query := `
		INSERT INTO fx_transactions(client_id, base_currency, target_currency, base_amount, target_amount, applied_rate, spread_bps) VALUES($1,$2,$3,$4,$5,$6,$7);`
	_, err := r.db.ExecContext(ctx,query, 
		tx.ClientID,
		tx.BaseCurrency,
		tx.TargetCurrency,
		tx.BaseAmount,
		tx.TargetAmount,
		tx.AppliedRate, 
		tx.SpreadBps,
	)
	if err != nil{
		return fmt.Errorf("failed to insert billing transaction: %w", err)
	}
	return nil
}