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
	if err := database.InitPostgres(); err != nil {
		log.Fatalf("Unable to connect to PostgreSQL: %v", err)
	}
	defer database.PgPool.Close()

	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	worker.StartLeaderElection()
	worker.StartWorker()

	fmt.Println("Server running on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
