package main

import (
	"fmt"
	"log"
	"net/http"
	"rinha-backend-2025/internal/worker"

	"rinha-backend-2025/internal/api"
	"rinha-backend-2025/internal/database"
)

func main() {
	database.InitRedis()

	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	worker.StartLeaderElection()
	worker.StartWorker()

	fmt.Println("Server running on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
