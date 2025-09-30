package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"github.com/segmentio/kafka-go"
)

var (
	redisClient *redis.Client
	kafkaWriter *kafka.Writer
	ctx         = context.Background()
	port        = getEnv("PORT", "3002")
)

type HealthResponse struct {
	Status    string    `json:"status"`
	Service   string    `json:"service"`
	Timestamp time.Time `json:"timestamp"`
}

type ClickEvent struct {
	ShortCode string    `json:"shortCode"`
	Timestamp time.Time `json:"timestamp"`
	UserAgent string    `json:"userAgent"`
	IP        string    `json:"ip"`
}

func main() {
	initRedis()
	initKafka()
	defer kafkaWriter.Close()

	router := mux.NewRouter()

	router.HandleFunc("/health", healthHandler).Methods("GET")
	router.HandleFunc("/{shortCode}", redirectHandler).Methods("GET")

	handler := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	}).Handler(router)

	log.Printf("[Redirect Service] Server starting on port %s\n", port)
	log.Printf("[Redirect Service] Connected to Redis at %s:%s\n",
		getEnv("REDIS_HOST", "localhost"),
		getEnv("REDIS_PORT", "6379"))
	log.Printf("[Redirect Service] Connected to Kafka at %s, topic: %s\n",
		getEnv("KAFKA_BROKERS", "localhost:9092"),
		getEnv("KAFKA_TOPIC", "url-clicks"))

	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatal(err)
	}
}

func initRedis() {
	redisHost := getEnv("REDIS_HOST", "localhost")
	redisPort := getEnv("REDIS_PORT", "6379")

	redisClient = redis.NewClient(&redis.Options{
		Addr:     redisHost + ":" + redisPort,
		Password: "",
		DB:       0,
	})

	_, err := redisClient.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
}

func initKafka() {
	brokers := strings.Split(getEnv("KAFKA_BROKERS", "localhost:9092"), ",")
	topic := getEnv("KAFKA_TOPIC", "url-clicks")

	kafkaWriter = &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		BatchSize:    1,
		RequiredAcks: kafka.RequireOne,
		Async:        true, // Асинхронная отправка для производительности
	}

	log.Printf("[Redirect Service] Kafka writer initialized")
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	_, err := redisClient.Ping(ctx).Result()
	status := "healthy"
	if err != nil {
		status = "unhealthy"
	}

	response := HealthResponse{
		Status:    status,
		Service:   "redirect-service",
		Timestamp: time.Now(),
	}
	respondJSON(w, http.StatusOK, response)
}

func redirectHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	shortCode := vars["shortCode"]

	// Получение оригинального URL из Redis
	originalURL, err := redisClient.Get(ctx, "url:"+shortCode).Result()
	if err == redis.Nil {
		log.Printf("[Redirect Service] Short code '%s' not found\n", shortCode)
		http.Error(w, "Short URL not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("[Redirect Service] Redis error: %v\n", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Асинхронная отправка события в Kafka
	go publishClickEvent(shortCode, r)

	log.Printf("[Redirect Service] Redirecting '%s' to %s\n", shortCode, originalURL)

	// Перенаправление
	http.Redirect(w, r, originalURL, http.StatusFound)
}

func publishClickEvent(shortCode string, r *http.Request) {
	event := ClickEvent{
		ShortCode: shortCode,
		Timestamp: time.Now(),
		UserAgent: r.UserAgent(),
		IP:        getIP(r),
	}

	jsonData, err := json.Marshal(event)
	if err != nil {
		log.Printf("[Redirect Service] Failed to marshal click event: %v\n", err)
		return
	}

	// Отправка сообщения в Kafka
	err = kafkaWriter.WriteMessages(context.Background(), kafka.Message{
		Key:   []byte(shortCode),
		Value: jsonData,
	})

	if err != nil {
		log.Printf("[Redirect Service] Failed to publish to Kafka: %v\n", err)
	} else {
		log.Printf("[Redirect Service] Published click event for '%s' to Kafka\n", shortCode)
	}
}

func getIP(r *http.Request) string {
	// Попытка получить реальный IP из заголовков
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		return forwarded
	}
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}
	return r.RemoteAddr
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
