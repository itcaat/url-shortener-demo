package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"log"
	"math/big"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	"github.com/itcaat/url-shortener-demo/pkg/tracing"
	"github.com/rs/cors"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"
)

var (
	redisClient *redis.Client
	ctx         = context.Background()
	port        = getEnv("PORT", "3001")
)

const (
	charset    = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	codeLength = 6
)

type ShortenRequest struct {
	URL string `json:"url"`
}

type ShortenResponse struct {
	ShortCode string `json:"shortCode"`
	ShortURL  string `json:"shortUrl"`
	Original  string `json:"originalUrl"`
}

type HealthResponse struct {
	Status    string    `json:"status"`
	Service   string    `json:"service"`
	Timestamp time.Time `json:"timestamp"`
}

func main() {
	// Initialize tracing (опционально, только если JAEGER_AGENT_HOST задан)
	if jaegerHost := os.Getenv("JAEGER_AGENT_HOST"); jaegerHost != "" {
		log.Println("[Shortener Service] Initializing distributed tracing...")
		tp, err := tracing.InitTracer("shortener-service")
		if err != nil {
			log.Printf("[Shortener Service] ⚠️  Failed to initialize tracer: %v", err)
		} else {
			defer tracing.Shutdown(context.Background(), tp)
			log.Println("[Shortener Service] ✅ Distributed tracing enabled")
		}
	} else {
		log.Println("[Shortener Service] ℹ️  Distributed tracing disabled (JAEGER_AGENT_HOST not set)")
	}

	initRedis()

	router := mux.NewRouter()

	// Add OpenTelemetry middleware
	router.Use(otelmux.Middleware("shortener-service"))

	router.HandleFunc("/health", healthHandler).Methods("GET")
	router.HandleFunc("/shorten", shortenHandler).Methods("POST")

	handler := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	}).Handler(router)

	// Graceful shutdown
	server := &http.Server{
		Addr:    ":" + port,
		Handler: handler,
	}

	go func() {
		log.Printf("[Shortener Service] Server starting on port %s\n", port)
		log.Printf("[Shortener Service] Connected to redis at %s:%s\n",
			getEnv("REDIS_HOST", "localhost"),
			getEnv("REDIS_PORT", "6379"))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("[Shortener Service] Shutting down gracefully...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("[Shortener Service] Server forced to shutdown:", err)
	}

	log.Println("[Shortener Service] Server exited")
}

func initRedis() {
	redisHost := getEnv("REDIS_HOST", "localhost")
	redisPort := getEnv("REDIS_PORT", "6379")

	redisClient = redis.NewClient(&redis.Options{
		Addr:     redisHost + ":" + redisPort,
		Password: "",
		DB:       0,
	})

	// Проверка соединения
	_, err := redisClient.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	// Проверка Redis
	_, err := redisClient.Ping(ctx).Result()
	status := "healthy"
	if err != nil {
		status = "unhealthy"
	}

	response := HealthResponse{
		Status:    status,
		Service:   "shortener-service",
		Timestamp: time.Now(),
	}
	respondJSON(w, http.StatusOK, response)
}

func shortenHandler(w http.ResponseWriter, r *http.Request) {
	var req ShortenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.URL == "" {
		respondError(w, http.StatusBadRequest, "URL is required")
		return
	}

	// Генерация короткого кода
	shortCode, err := generateShortCode()
	if err != nil {
		log.Printf("[Shortener Service] Error generating short code: %v\n", err)
		respondError(w, http.StatusInternalServerError, "Failed to generate short code")
		return
	}

	// Проверка уникальности
	for {
		exists, err := redisClient.Exists(ctx, "url:"+shortCode).Result()
		if err != nil {
			log.Printf("[Shortener Service] Redis error: %v\n", err)
			respondError(w, http.StatusInternalServerError, "Database error")
			return
		}
		if exists == 0 {
			break
		}
		shortCode, _ = generateShortCode()
	}

	// Сохранение в Redis
	err = redisClient.Set(ctx, "url:"+shortCode, req.URL, 0).Err()
	if err != nil {
		log.Printf("[Shortener Service] Failed to save to Redis: %v\n", err)
		respondError(w, http.StatusInternalServerError, "Failed to save URL")
		return
	}

	log.Printf("[Shortener Service] Created short code '%s' for URL: %s\n", shortCode, req.URL)

	response := ShortenResponse{
		ShortCode: shortCode,
		ShortURL:  "http://localhost:3002/" + shortCode,
		Original:  req.URL,
	}

	respondJSON(w, http.StatusCreated, response)
}

func generateShortCode() (string, error) {
	result := make([]byte, codeLength)
	charsetLen := big.NewInt(int64(len(charset)))

	for i := 0; i < codeLength; i++ {
		num, err := rand.Int(rand.Reader, charsetLen)
		if err != nil {
			return "", err
		}
		result[i] = charset[num.Int64()]
	}

	return string(result), nil
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
