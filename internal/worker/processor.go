package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"rinha-backend-2025/internal/core"
	"rinha-backend-2025/internal/database"
)

func processPayment(ctx context.Context, payment core.PaymentRequest) error {
	health, err := RetrieveHealthStates(ctx)
	if err != nil {
		return fmt.Errorf("failed to retrieve health states: %w", err)
	}

	processorURL := os.Getenv("PROCESSOR_DEFAULT_URL")
	processorType := "DEFAULT"
	status := "PROCESSED_DEFAULT"
	if health.DefaultProcessor.Failing || health.DefaultProcessor.MinResponseTime > health.FallBackProcessor.MinResponseTime+50 {
		processorURL = os.Getenv("PROCESSOR_FALLBACK_URL")
		processorType = "FALLBACK"
		status = "PROCESSED_FALLBACK"
	}

	requestedAt := time.Now().UTC().Format(time.RFC3339Nano)

	body := map[string]interface{}{
		"correlationId": payment.CorrelationID,
		"amount":        payment.Amount,
		"requestedAt":   time.Now().UTC().Format(time.RFC3339Nano),
	}

	jsonBody, _ := json.Marshal(body)

	resp, err := http.Post(processorURL+"/payments", "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		return err
	} else {
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return fmt.Errorf("failed to process payment: %s", resp.Status)
		}
	}

	processedPayment := core.ProcessedPayment{
		CorrelationID: payment.CorrelationID,
		Amount:        payment.Amount,
		Status:        status,
		Processor:     processorType,
		CreatedAt:     requestedAt,
	}

	paymentData, err := json.Marshal(processedPayment)
	if err != nil {
		return fmt.Errorf("failed to marshal payment data: %w", err)
	}

	err = database.Rdb.HSet(database.RedisCtx, "payments", payment.CorrelationID, paymentData).Err()
	if err != nil {
		return fmt.Errorf("failed to save payment in redis: %w", err)
	}

	return nil
}
