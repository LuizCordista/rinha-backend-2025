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

	// Prepare request to processor
	body := map[string]interface{}{
		"correlationId": payment.CorrelationID,
		"amount":        payment.Amount,
		"requestedAt":   time.Now().UTC().Format(time.RFC3339Nano),
	}

	jsonBody, _ := json.Marshal(body)

	resp, err := http.Post(processorURL+"/payments", "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		status = "FAILED"
		processorType = "DEFAULT"
	} else {
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			status = "FAILED"
		}
	}

	_, err = database.PgPool.Exec(ctx, `
		INSERT INTO payments (correlation_id, amount, status, processor)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (correlation_id) DO UPDATE SET status = $3, processor = $4
	`, payment.CorrelationID, payment.Amount, status, processorType)
	if err != nil {
		return fmt.Errorf("failed to update payment in postgres: %w", err)
	}

	return nil
}
