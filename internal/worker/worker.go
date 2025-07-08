package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"rinha-backend-2025/internal/core"
	"rinha-backend-2025/internal/database"
)

func StartWorker() {
	workerID := os.Getenv("INSTANCE_ID")
	if workerID == "" {
		workerID = fmt.Sprintf("worker-%d", time.Now().UnixNano())
	}
	processingQueue := "payments_processing:" + workerID

	go func() {
		for {
			res, err := database.Rdb.RPopLPush(context.Background(), "payments_pending", processingQueue).Result()
			if err != nil {
				if err.Error() != "redis: nil" {
					fmt.Println("Error moving payment to processing queue:", err)
				}
				time.Sleep(1 * time.Second)
				continue
			}

			var payment core.PaymentRequest

			if err := json.Unmarshal([]byte(res), &payment); err != nil {
				fmt.Printf("[Worker %s] Failed to unmarshal payment: %v\n", workerID, err)
				continue
			}

			if err := processPayment(context.Background(), payment); err != nil {
				fmt.Printf("[Worker %s] Failed to process payment: %v\n", workerID, err)
			} else {
				fmt.Printf("[Worker %s] Payment processed: %s\n", workerID, payment.CorrelationID)
			}
		}
	}()
}
