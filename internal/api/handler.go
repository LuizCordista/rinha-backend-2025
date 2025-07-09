package api

import (
	"encoding/json"
	"net/http"
	"time"

	"rinha-backend-2025/internal/core"
	"rinha-backend-2025/internal/database"
)

func PaymentsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var p core.PaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	data, err := json.Marshal(p)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if err := database.Rdb.LPush(database.RedisCtx, "payments_pending", data).Err(); err != nil {
		http.Error(w, "Failed to queue payment", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

func PaymentsSummaryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	if from == "" || to == "" {
		http.Error(w, "Missing 'from' or 'to' query parameter", http.StatusBadRequest)
		return
	}

	// Parse time
	fromTime, err := time.Parse(time.RFC3339Nano, from)
	if err != nil {
		http.Error(w, "Invalid 'from' datetime format", http.StatusBadRequest)
		return
	}
	toTime, err := time.Parse(time.RFC3339Nano, to)
	if err != nil {
		http.Error(w, "Invalid 'to' datetime format", http.StatusBadRequest)
		return
	}

	// Buscar todos os pagamentos do Redis
	paymentsData, err := database.Rdb.HGetAll(database.RedisCtx, "payments").Result()
	if err != nil {
		http.Error(w, "Failed to retrieve payments from Redis", http.StatusInternalServerError)
		return
	}

	var defaultCount int
	var defaultSum float64
	var fallbackCount int
	var fallbackSum float64

	// Processar cada pagamento
	for _, paymentDataStr := range paymentsData {
		var payment core.ProcessedPayment
		if err := json.Unmarshal([]byte(paymentDataStr), &payment); err != nil {
			continue // Skip invalid payments
		}

		// Parse created_at time
		createdAt, err := time.Parse(time.RFC3339Nano, payment.CreatedAt)
		if err != nil {
			continue // Skip payments with invalid timestamps
		}

		// Check if payment is within time range
		if createdAt.Before(fromTime) || createdAt.After(toTime) {
			continue
		}

		// Count and sum by processor type and status
		if payment.Processor == "DEFAULT" && payment.Status == "PROCESSED_DEFAULT" {
			defaultCount++
			defaultSum += payment.Amount
		} else if payment.Processor == "FALLBACK" && payment.Status == "PROCESSED_FALLBACK" {
			fallbackCount++
			fallbackSum += payment.Amount
		}
	}

	resp := core.PaymentsSummaryResponse{
		Default: core.PaymentsSummary{
			TotalRequests: defaultCount,
			TotalAmount:   core.RoundedFloat(defaultSum),
		},
		Fallback: core.PaymentsSummary{
			TotalRequests: fallbackCount,
			TotalAmount:   core.RoundedFloat(fallbackSum),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func PurgePaymentsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// Deletar todos os pagamentos do Redis
	err := database.Rdb.Del(database.RedisCtx, "payments").Err()
	if err != nil {
		http.Error(w, "Failed to purge payments", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
