package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/itcaat/url-shortener-demo/pkg/tracing"
	"github.com/rs/cors"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"
)

var (
	shortenerServiceURL = getEnv("SHORTENER_SERVICE_URL", "http://localhost:3001")
	analyticsServiceURL = getEnv("ANALYTICS_SERVICE_URL", "http://localhost:3003")
	port                = getEnv("PORT", "3000")
)

type HealthResponse struct {
	Status    string    `json:"status"`
	Service   string    `json:"service"`
	Timestamp time.Time `json:"timestamp"`
}

type ServiceInfo struct {
	Name   string      `json:"name"`
	URL    string      `json:"url"`
	Status string      `json:"status"`
	Data   interface{} `json:"data,omitempty"`
	Error  string      `json:"error,omitempty"`
}

type InfoResponse struct {
	Gateway   string        `json:"gateway"`
	Version   string        `json:"version"`
	Services  []ServiceInfo `json:"services"`
	Timestamp time.Time     `json:"timestamp"`
}

func main() {
	// Initialize tracing (опционально, только если JAEGER_AGENT_HOST задан)
	if jaegerHost := os.Getenv("JAEGER_AGENT_HOST"); jaegerHost != "" {
		log.Println("[API Gateway] Initializing distributed tracing...")
		tp, err := tracing.InitTracer("api-gateway")
		if err != nil {
			log.Printf("[API Gateway] ⚠️  Failed to initialize tracer: %v", err)
		} else {
			defer tracing.Shutdown(context.Background(), tp)
			log.Println("[API Gateway] ✅ Distributed tracing enabled")
		}
	} else {
		log.Println("[API Gateway] ℹ️  Distributed tracing disabled (JAEGER_AGENT_HOST not set)")
	}

	router := mux.NewRouter()

	// Add OpenTelemetry middleware for automatic tracing
	router.Use(otelmux.Middleware("api-gateway"))

	// Health check
	router.HandleFunc("/health", healthHandler).Methods("GET")

	// API routes
	router.HandleFunc("/api/shorten", shortenHandler).Methods("POST")
	router.HandleFunc("/api/stats/{shortCode}", statsHandler).Methods("GET")
	router.HandleFunc("/api/stats", allStatsHandler).Methods("GET")
	router.HandleFunc("/api/info", infoHandler).Methods("GET")

	// CORS
	handler := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	}).Handler(router)

	// Graceful shutdown
	server := &http.Server{
		Addr:    ":" + port,
		Handler: handler,
	}

	go func() {
		log.Printf("[API Gateway] Server starting. Port %s\n", port)
		log.Printf("[API Gateway] Shortener Service: %s\n", shortenerServiceURL)
		log.Printf("[API Gateway] Analytics Service: %s\n", analyticsServiceURL)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("[API Gateway] Shutting down gracefully...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("[API Gateway] Server forced to shutdown:", err)
	}

	log.Println("[API Gateway] Server exited")
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	response := HealthResponse{
		Status:    "healthy",
		Service:   "api-gateway",
		Timestamp: time.Now(),
	}
	respondJSON(w, http.StatusOK, response)
}

func shortenHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("[API Gateway] Proxying shorten request to %s\n", shortenerServiceURL)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Failed to read request body")
		return
	}

	resp, err := http.Post(shortenerServiceURL+"/shorten", "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("[API Gateway] Error proxying to shortener-service: %v\n", err)
		respondError(w, http.StatusServiceUnavailable, "Failed to connect to shortener service")
		return
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to read response")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	w.Write(respBody)
}

func statsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	shortCode := vars["shortCode"]

	log.Printf("[API Gateway] Proxying stats request for %s to %s\n", shortCode, analyticsServiceURL)

	resp, err := http.Get(fmt.Sprintf("%s/stats/%s", analyticsServiceURL, shortCode))
	if err != nil {
		log.Printf("[API Gateway] Error proxying to analytics-service: %v\n", err)
		respondError(w, http.StatusServiceUnavailable, "Failed to connect to analytics service")
		return
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to read response")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	w.Write(respBody)
}

func allStatsHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("[API Gateway] Proxying all stats request to %s\n", analyticsServiceURL)

	resp, err := http.Get(analyticsServiceURL + "/stats")
	if err != nil {
		log.Printf("[API Gateway] Error proxying to analytics-service: %v\n", err)
		respondError(w, http.StatusServiceUnavailable, "Failed to connect to analytics service")
		return
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to read response")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	w.Write(respBody)
}

func infoHandler(w http.ResponseWriter, r *http.Request) {
	services := []ServiceInfo{
		checkService("shortener-service", shortenerServiceURL),
		checkService("analytics-service", analyticsServiceURL),
	}

	response := InfoResponse{
		Gateway:   "api-gateway",
		Version:   "1.0.0",
		Services:  services,
		Timestamp: time.Now(),
	}

	respondJSON(w, http.StatusOK, response)
}

func checkService(name, url string) ServiceInfo {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(url + "/health")
	if err != nil {
		return ServiceInfo{
			Name:   name,
			URL:    url,
			Status: "unhealthy",
			Error:  err.Error(),
		}
	}
	defer resp.Body.Close()

	var data interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err == nil {
		return ServiceInfo{
			Name:   name,
			URL:    url,
			Status: "healthy",
			Data:   data,
		}
	}

	return ServiceInfo{
		Name:   name,
		URL:    url,
		Status: "healthy",
	}
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
