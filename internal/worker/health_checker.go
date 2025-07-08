package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"rinha-backend-2025/internal/core"
	"sync/atomic"
	"time"

	"rinha-backend-2025/internal/database"
)

var isLeader atomic.Bool

func StartLeaderElection() {
	instanceID := os.Getenv("INSTANCE_ID")
	if instanceID == "" {
		instanceID = fmt.Sprintf("instance-%d", time.Now().UnixNano())
	}

	go func() {
		for {
			acquired, err := database.Rdb.SetNX(database.RedisCtx, "rinha-leader-lock", instanceID, 10*time.Second).Result()
			if err != nil {
				fmt.Println("Redis error during leader election:", err)
			}

			if acquired {
				fmt.Println("Became leader:", instanceID)

				isLeader.Store(true)

				go renewLeaderLock(instanceID)
				go healthChecker()

				return
			} else {
				isLeader.Store(false)
			}

			time.Sleep(3 * time.Second)
		}
	}()
}

func renewLeaderLock(instanceID string) {
	ticker := time.NewTicker(8 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		val, err := database.Rdb.Get(database.RedisCtx, "rinha-leader-lock").Result()
		if err == nil && val == instanceID {
			database.Rdb.Set(database.RedisCtx, "rinha-leader-lock", instanceID, 10*time.Second)
		} else {
			fmt.Println("Lost leadership")
			isLeader.Store(false)
			return
		}
	}
}

func healthChecker() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	processorURLs := []string{
		os.Getenv("PROCESSOR_DEFAULT_URL"),
		os.Getenv("PROCESSOR_FALLBACK_URL"),
	}

	redisKeys := []string{"health:default", "health:fallback"}

	for {
		<-ticker.C
		for i, url := range processorURLs {
			go func(i int, url string) {
				resp, err := http.Get(url + "/payments/service-health")
				if err != nil {
					fmt.Printf("Health check failed for %s: %v\n", url, err)
					return
				}
				defer resp.Body.Close()

				if resp.StatusCode != http.StatusOK {
					fmt.Printf("Health check non-200 for %s: %d\n", url, resp.StatusCode)
					return
				}

				var hr core.HealthResponse

				if err := json.NewDecoder(resp.Body).Decode(&hr); err != nil {
					fmt.Printf("Failed to decode health response from %s: %v\n", url, err)
					return
				}

				fmt.Printf("Health for %s: failing=%v, minResponseTime=%d\n", url, hr.Failing, hr.MinResponseTime)

				data, err := json.Marshal(hr)
				if err != nil {
					fmt.Printf("Failed to marshal health response for %s: %v\n", url, err)
					return
				}

				// Use a fresh context for Redis to avoid context timeout propagation
				redisCtx := context.Background()
				if err := database.Rdb.Set(redisCtx, redisKeys[i], data, 0).Err(); err != nil {
					fmt.Printf("Failed to save health state for %s in Redis: %v\n", url, err)
				}
			}(i, url)
		}
	}
}

func RetrieveHealthStates(ctx context.Context) (*core.HealthManager, error) {
	defaultKey := "health:default"
	fallbackKey := "health:fallback"

	defaultVal, err := database.Rdb.Get(ctx, defaultKey).Result()
	if err != nil {
		return nil, err
	}
	fallbackVal, err := database.Rdb.Get(ctx, fallbackKey).Result()
	if err != nil {
		return nil, err
	}

	var defaultHealth core.HealthResponse
	var fallbackHealth core.HealthResponse

	if err := json.Unmarshal([]byte(defaultVal), &defaultHealth); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(fallbackVal), &fallbackHealth); err != nil {
		return nil, err
	}

	return &core.HealthManager{
		DefaultProcessor:  defaultHealth,
		FallBackProcessor: fallbackHealth,
	}, nil
}
