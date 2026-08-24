package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"fx-gateway/internal/usecase"
)

type PriceRequest struct {
	ClientID       string `json:"client_id"`
	BaseAmount     int64  `json:"base_amount"`     
	BaseCurrency   string `json:"base_currency"`  
	TargetCurrency string `json:"target_currency"` 
}

type PriceResponse struct {
	TargetAmount int64 `json:"target_amount"`
	AppliedRate  int64 `json:"applied_rate"`  
}

type FXGatewayUseCaseAPI interface {
	Execute(ctx context.Context, req usecase.ConvertAndBillingRequest) (*usecase.ConvertAndBillingResponse, error)
}

type ValuteHandler struct{
	usecase FXGatewayUseCaseAPI
}

func NewValuteHandler(uc FXGatewayUseCaseAPI) *ValuteHandler{
	return &ValuteHandler{usecase: uc}
}

func (h *ValuteHandler) GetPrice(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	r.Body = http.MaxBytesReader(w, r.Body, 1048576)
	defer r.Body.Close()

	var reqBody PriceRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err !=nil {
		slog.Warn("handler: failed to decode json request", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		_,_ = w.Write([]byte(`"error": "invalid json body"`))
		return
	}
	if reqBody.ClientID == "" || reqBody.BaseCurrency == "" || reqBody.TargetCurrency == "" || reqBody.BaseAmount <= 0 {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"error":"missing required fields or invalid amount"}`))
		return
	}
	ucReq := usecase.ConvertAndBillingRequest{
		ClientID:       reqBody.ClientID,
		BaseAmount:     reqBody.BaseAmount,
		BaseCurrency:   reqBody.BaseCurrency,
		TargetCurrency: reqBody.TargetCurrency,
	}
	ucResp, err := h.usecase.Execute(r.Context(), ucReq)
	if err != nil {
		slog.Error("handler: usecase execution failed", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error": "internal currency conversion error"}`))
		return
	}
	respBody := PriceResponse{
		TargetAmount: ucResp.TargetAmount,
		AppliedRate:  ucResp.AppliedRate,
	}
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(respBody); err != nil {
		slog.Error("handler: failed to encode json response", "error", err)
	}
}