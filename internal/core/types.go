package core

import (
	"encoding/json"
	"math"
)

type PaymentRequest struct {
	CorrelationID string  `json:"correlationId"`
	Amount        float64 `json:"amount"`
}

type HealthResponse struct {
	Failing         bool `json:"failing"`
	MinResponseTime int  `json:"minResponseTime"`
}

type HealthManager struct {
	DefaultProcessor  HealthResponse
	FallBackProcessor HealthResponse
}

type RoundedFloat float64

func (r RoundedFloat) MarshalJSON() ([]byte, error) {
	rounded := math.Round(float64(r)*10) / 10
	return json.Marshal(rounded)
}

type PaymentsSummary struct {
	TotalRequests int          `json:"totalRequests"`
	TotalAmount   RoundedFloat `json:"totalAmount"`
}

type PaymentsSummaryResponse struct {
	Default  PaymentsSummary `json:"default"`
	Fallback PaymentsSummary `json:"fallback"`
}

type ProcessedPayment struct {
	CorrelationID string  `json:"correlationId"`
	Amount        float64 `json:"amount"`
	Status        string  `json:"status"`
	Processor     string  `json:"processor"`
	CreatedAt     string  `json:"createdAt"`
}
