package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"github.com/segmentio/kafka-go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	mongoClient *mongo.Client
	collection  *mongo.Collection
	kafkaReader *kafka.Reader
	ctx         = context.Background()
	port        = getEnv("PORT", "3003")
)

type ClickEvent struct {
	ShortCode string    `bson:"shortCode" json:"shortCode"`
	Timestamp time.Time `bson:"timestamp" json:"timestamp"`
	UserAgent string    `bson:"userAgent" json:"userAgent"`
	IP        string    `bson:"ip" json:"ip"`
}

type StatsResponse struct {
	ShortCode   string     `json:"shortCode"`
	TotalClicks int64      `json:"totalClicks"`
	LastClick   *time.Time `json:"lastClick,omitempty"`
}

type AllStatsResponse struct {
	Stats []StatsResponse `json:"stats"`
	Total int             `json:"total"`
}

type HealthResponse struct {
	Status    string    `json:"status"`
	Service   string    `json:"service"`
	Timestamp time.Time `json:"timestamp"`
}

func main() {
	initMongoDB()
	defer mongoClient.Disconnect(ctx)

	initKafka()
	defer kafkaReader.Close()

	// Запуск Kafka consumer в отдельной горутине
	go consumeKafkaMessages()

	// HTTP сервер для статистики
	router := mux.NewRouter()

	router.HandleFunc("/health", healthHandler).Methods("GET")
	router.HandleFunc("/stats/{shortCode}", statsHandler).Methods("GET")
	router.HandleFunc("/stats", allStatsHandler).Methods("GET")

	handler := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	}).Handler(router)

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	server := &http.Server{
		Addr:    ":" + port,
		Handler: handler,
	}

	go func() {
		log.Printf("[Analytics Service] Server starting on port %s\n", port)
		log.Printf("[Analytics Service] Connected to MongoDB: %s\n", getEnv("MONGODB_URI", "mongodb://localhost:27017/analytics"))
		log.Printf("[Analytics Service] Consuming from Kafka topic: %s\n", getEnv("KAFKA_TOPIC", "url-clicks"))

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	<-sigChan
	log.Println("[Analytics Service] Shutting down gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("[Analytics Service] Error during shutdown: %v\n", err)
	}
}

func initMongoDB() {
	mongoURI := getEnv("MONGODB_URI", "mongodb://localhost:27017/analytics")

	clientOptions := options.Client().ApplyURI(mongoURI)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}

	err = client.Ping(ctx, nil)
	if err != nil {
		log.Fatalf("Failed to ping MongoDB: %v", err)
	}

	mongoClient = client
	collection = client.Database("analytics").Collection("clicks")

	// Создание индекса для shortCode
	indexModel := mongo.IndexModel{
		Keys: bson.D{{Key: "shortCode", Value: 1}},
	}
	_, err = collection.Indexes().CreateOne(ctx, indexModel)
	if err != nil {
		log.Printf("Warning: Failed to create index: %v", err)
	}
}

func initKafka() {
	brokers := strings.Split(getEnv("KAFKA_BROKERS", "localhost:9092"), ",")
	topic := getEnv("KAFKA_TOPIC", "url-clicks")
	groupID := getEnv("KAFKA_GROUP_ID", "analytics-consumer-group")

	kafkaReader = kafka.NewReader(kafka.ReaderConfig{
		Brokers:        brokers,
		Topic:          topic,
		GroupID:        groupID,
		MinBytes:       10e3, // 10KB
		MaxBytes:       10e6, // 10MB
		CommitInterval: time.Second,
		StartOffset:    kafka.LastOffset,
	})

	log.Printf("[Analytics Service] Kafka consumer initialized")
}

func consumeKafkaMessages() {
	log.Println("[Analytics Service] Starting Kafka consumer...")

	for {
		msg, err := kafkaReader.ReadMessage(context.Background())
		if err != nil {
			log.Printf("[Analytics Service] Error reading message: %v\n", err)
			time.Sleep(time.Second)
			continue
		}

		var event ClickEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			log.Printf("[Analytics Service] Failed to unmarshal message: %v\n", err)
			continue
		}

		// Сохранение в MongoDB
		_, err = collection.InsertOne(context.Background(), event)
		if err != nil {
			log.Printf("[Analytics Service] Failed to insert click event: %v\n", err)
			continue
		}

		log.Printf("[Analytics Service] Processed click event for '%s' from Kafka (offset: %d)\n",
			event.ShortCode, msg.Offset)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	err := mongoClient.Ping(ctx, nil)
	status := "healthy"
	if err != nil {
		status = "unhealthy"
	}

	response := HealthResponse{
		Status:    status,
		Service:   "analytics-service",
		Timestamp: time.Now(),
	}
	respondJSON(w, http.StatusOK, response)
}

func statsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	shortCode := vars["shortCode"]

	count, err := collection.CountDocuments(ctx, bson.M{"shortCode": shortCode})
	if err != nil {
		log.Printf("[Analytics Service] Failed to count documents: %v\n", err)
		respondError(w, http.StatusInternalServerError, "Failed to get statistics")
		return
	}

	response := StatsResponse{
		ShortCode:   shortCode,
		TotalClicks: count,
	}

	// Получение последнего клика
	opts := options.FindOne().SetSort(bson.D{{Key: "timestamp", Value: -1}})
	var lastClick ClickEvent
	err = collection.FindOne(ctx, bson.M{"shortCode": shortCode}, opts).Decode(&lastClick)
	if err == nil {
		response.LastClick = &lastClick.Timestamp
	}

	respondJSON(w, http.StatusOK, response)
}

func allStatsHandler(w http.ResponseWriter, r *http.Request) {
	pipeline := mongo.Pipeline{
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$shortCode"},
			{Key: "totalClicks", Value: bson.D{{Key: "$sum", Value: 1}}},
			{Key: "lastClick", Value: bson.D{{Key: "$max", Value: "$timestamp"}}},
		}}},
		{{Key: "$sort", Value: bson.D{{Key: "totalClicks", Value: -1}}}},
	}

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		log.Printf("[Analytics Service] Failed to aggregate stats: %v\n", err)
		respondError(w, http.StatusInternalServerError, "Failed to get statistics")
		return
	}
	defer cursor.Close(ctx)

	var stats []StatsResponse
	for cursor.Next(ctx) {
		var result struct {
			ID          string    `bson:"_id"`
			TotalClicks int64     `bson:"totalClicks"`
			LastClick   time.Time `bson:"lastClick"`
		}
		if err := cursor.Decode(&result); err != nil {
			log.Printf("[Analytics Service] Failed to decode result: %v\n", err)
			continue
		}

		stats = append(stats, StatsResponse{
			ShortCode:   result.ID,
			TotalClicks: result.TotalClicks,
			LastClick:   &result.LastClick,
		})
	}

	if stats == nil {
		stats = []StatsResponse{}
	}

	response := AllStatsResponse{
		Stats: stats,
		Total: len(stats),
	}

	respondJSON(w, http.StatusOK, response)
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
