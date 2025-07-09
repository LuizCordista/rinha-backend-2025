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

	from, to, useFilter, err := parseTimeRange(r)
	if err != nil {
		if err == http.ErrMissingFile {
			http.Error(w, "Both 'from' and 'to' query parameters are required, or omit both for all data", http.StatusBadRequest)
		} else {
			http.Error(w, "Invalid datetime format", http.StatusBadRequest)
		}
		return
	}

	paymentsData, err := database.Rdb.HGetAll(database.RedisCtx, "payments").Result()
	if err != nil {
		http.Error(w, "Failed to retrieve payments from Redis", http.StatusInternalServerError)
		return
	}

	resp := summarizePayments(paymentsData, from, to, useFilter)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func PurgePaymentsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	err := database.Rdb.Del(database.RedisCtx, "payments").Err()
	if err != nil {
		http.Error(w, "Failed to purge payments", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func parseTimeRange(r *http.Request) (from, to time.Time, useFilter bool, err error) {
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")

	if fromStr == "" && toStr == "" {
		return
	}
	if fromStr == "" || toStr == "" {
		err = http.ErrMissingFile
		return
	}
	from, err = time.Parse(time.RFC3339Nano, fromStr)
	if err != nil {
		return
	}
	to, err = time.Parse(time.RFC3339Nano, toStr)
	if err != nil {
		return
	}
	useFilter = true
	return
}

func summarizePayments(paymentsData map[string]string, from, to time.Time, useFilter bool) core.PaymentsSummaryResponse {
	var defaultCount, fallbackCount int
	var defaultSum, fallbackSum float64

	for _, paymentDataStr := range paymentsData {
		var payment core.ProcessedPayment
		if err := json.Unmarshal([]byte(paymentDataStr), &payment); err != nil {
			continue
		}
		if useFilter {
			createdAt, err := time.Parse(time.RFC3339Nano, payment.CreatedAt)
			if err != nil || createdAt.Before(from) || createdAt.After(to) {
				continue
			}
		}
		if payment.Processor == "DEFAULT" && payment.Status == "PROCESSED_DEFAULT" {
			defaultCount++
			defaultSum += payment.Amount
		} else if payment.Processor == "FALLBACK" && payment.Status == "PROCESSED_FALLBACK" {
			fallbackCount++
			fallbackSum += payment.Amount
		}
	}
	return core.PaymentsSummaryResponse{
		Default: core.PaymentsSummary{
			TotalRequests: defaultCount,
			TotalAmount:   core.RoundedFloat(defaultSum),
		},
		Fallback: core.PaymentsSummary{
			TotalRequests: fallbackCount,
			TotalAmount:   core.RoundedFloat(fallbackSum),
		},
	}
}
