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

	ctx := database.PgCtx
	// Query for DEFAULT processor
	row := database.PgPool.QueryRow(ctx, `
		SELECT COUNT(*), COALESCE(SUM(amount),0)
		FROM payments
		WHERE processor = 'DEFAULT' AND status IN ('PROCESSED_DEFAULT')
		AND created_at BETWEEN $1 AND $2
	`, fromTime, toTime)
	var defaultCount int
	var defaultSum float64
	if err := row.Scan(&defaultCount, &defaultSum); err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}
	// Query for FALLBACK processor
	row = database.PgPool.QueryRow(ctx, `
		SELECT COUNT(*), COALESCE(SUM(amount),0)
		FROM payments
		WHERE processor = 'FALLBACK' AND status IN ('PROCESSED_FALLBACK')
		AND created_at BETWEEN $1 AND $2
	`, fromTime, toTime)
	var fallbackCount int
	var fallbackSum float64
	if err := row.Scan(&fallbackCount, &fallbackSum); err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}

	resp := core.PaymentsSummaryResponse{
		Default: core.PaymentsSummary{
			TotalRequests: defaultCount,
			TotalAmount:   defaultSum,
		},
		Fallback: core.PaymentsSummary{
			TotalRequests: fallbackCount,
			TotalAmount:   fallbackSum,
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

	_, err := database.PgPool.Exec(database.PgCtx, "DELETE FROM payments")
	if err != nil {
		http.Error(w, "Failed to purge payments", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
