package domain

import "errors"

var (
	ErrInvalidAmount   = errors.New("base amount must be strictly greater than zero")
	ErrInvalidCurrency = errors.New("currency code must be exactly 3 uppercase letters")
	ErrMissingClientID = errors.New("client_id cannot be empty")
)