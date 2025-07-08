package api

import (
	"net/http"
)

func RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/payments", PaymentsHandler)
	mux.HandleFunc("/payments-summary", PaymentsSummaryHandler)
	mux.HandleFunc("/purge-payments", PurgePaymentsHandler)
}
