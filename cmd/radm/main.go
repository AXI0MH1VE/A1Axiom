package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"anomaly"
	"internal/config"
	"internal/monetization"
	"internal/ratelimit"
	"internal/validation"
)

// Response represents the API response structure.
type Response struct {
	IsAnomaly   bool    `json:"is_anomaly"`
	ZScore      float64 `json:"z_score"`
	Timestamp   int64   `json:"timestamp"`
	Value       float64 `json:"value"`
	ProcessingNS int64   `json:"processing_ns"`
	Price       float64 `json:"price,omitempty"`
}

// ErrorResponse represents an error response.
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

var (
	detector   *anomaly.AnomalyDetector
	monTracker *monetization.MonetizationTracker
	validator  *validation.DataPointValidator
	rateLimit  *ratelimit.RateLimiter
	cfg        *config.Config
)

func main() {
	// Load configuration
	var err error
	cfg, err = config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	// Initialize components
	initializeComponents()

	// Setup HTTP server
	router := setupRouter()

	// Setup graceful shutdown
	setupGracefulShutdown(router)

	// Start server
	serverAddr := cfg.Server.Host + ":" + cfg.Server.Port
	log.Printf("Starting RADM server on %s", serverAddr)
	log.Printf("Configuration: WindowSize=%d, Threshold=%.2f",
		cfg.Detector.WindowSize, cfg.Detector.Threshold)

	if err := http.ListenAndServe(serverAddr, router); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed to start: %v", err)
	}
}

// initializeComponents initializes all the core components.
func initializeComponents() {
	// Initialize anomaly detector
	detector = anomaly.NewDetector(cfg.Detector.WindowSize, cfg.Detector.Threshold)

	// Initialize monetization tracker
	if cfg.Monetization.Enabled {
		monConfig := monetization.Config{
			BasePrice:            cfg.Monetization.BasePrice,
			ComplexityMultiplier: cfg.Monetization.ComplexityMultiplier,
			OutputFile:           cfg.Monetization.OutputFile,
		}
		monTracker = monetization.NewTracker(monConfig)
	}

	// Initialize validator
	if cfg.Validation.Enabled {
		valConfig := validation.Config{
			MaxValue:      cfg.Validation.MaxValue,
			MinValue:      cfg.Validation.MinValue,
			MaxTimestamp:  cfg.Validation.MaxTimestamp,
			MinTimestamp:  cfg.Validation.MinTimestamp,
			AllowedSource: cfg.Validation.AllowedSource,
		}
		validator = validation.NewDataPointValidator(valConfig)
	}

	// Initialize rate limiter
	if cfg.RateLimit.Enabled {
		rateLimit = ratelimit.NewRateLimiter(cfg.RateLimit.RequestsPerSecond, cfg.RateLimit.BurstSize)
	}
}

// setupRouter configures the HTTP router with all endpoints.
func setupRouter() *chi.Mux {
	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)

	// Health check endpoint (Protocol β-RedTeam/Kubernetes)
	r.Get("/healthz", healthCheckHandler)
	r.Get("/readyz", readyCheckHandler)

	// Metrics endpoint
	r.Get("/metrics", metricsHandler)

	// Main ingestion endpoint with rate limiting
	r.With(rateLimitMiddleware).Post("/api/v1/data/ingest", ingestHandler)

	return r
}

// rateLimitMiddleware implements Protocol α-IngressGuard rate limiting.
func rateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if rateLimit != nil {
			allowed := rateLimit.Allow()
			if !allowed {
				log.Printf("Rate limit exceeded for IP: %s", getClientIP(r))
				writeErrorResponse(w, http.StatusTooManyRequests, "RATE_LIMIT_EXCEEDED",
					"Rate limit exceeded. Please try again later.")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// Note: Allow method is implemented in the ratelimit package

// healthCheckHandler handles health check requests.
func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// readyCheckHandler handles readiness check requests.
func readyCheckHandler(w http.ResponseWriter, r *http.Request) {
	// Check if all components are ready
	if detector == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("NOT_READY"))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("READY"))
}

// metricsHandler provides system metrics.
func metricsHandler(w http.ResponseWriter, r *http.Request) {
	stats := map[string]interface{}{
		"detector_stats":   getDetectorStats(),
		"rate_limit_stats": getRateLimitStats(),
		"monetization_stats": getMonetizationStats(),
		"uptime_seconds":   time.Since(startTime).Seconds(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// ingestHandler handles data ingestion requests.
func ingestHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Parse request body
	var reqBody map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "INVALID_JSON",
			"Invalid JSON in request body")
		return
	}

	// Extract timestamp and value
	timestamp, ok := reqBody["timestamp"].(float64)
	if !ok {
		writeErrorResponse(w, http.StatusBadRequest, "MISSING_TIMESTAMP",
			"timestamp field is required and must be a number")
		return
	}

	value, ok := reqBody["value"].(float64)
	if !ok {
		writeErrorResponse(w, http.StatusBadRequest, "MISSING_VALUE",
			"value field is required and must be a number")
		return
	}

	// Validate input (Protocol α-IngressGuard)
	if validator != nil {
		if err := validator.ValidateDataPoint(int64(timestamp), value, getClientIP(r)); err != nil {
			log.Printf("Validation failed for IP %s: %v", getClientIP(r), err)
			writeErrorResponse(w, http.StatusBadRequest, "VALIDATION_FAILED", err.Error())
			return
		}
	}

	// Process data
	dp := anomaly.DataPoint{
		Timestamp: int64(timestamp),
		Value:     value,
	}

	isAnomaly, zScore, err := detector.ProcessData(dp)
	if err != nil {
		log.Printf("Error processing data: %v", err)
		writeErrorResponse(w, http.StatusInternalServerError, "PROCESSING_ERROR",
			"Error processing data point")
		return
	}

	// Record monetization data (Axiom A-4 Hook)
	latencyNS := time.Since(start).Nanoseconds()
	price := 0.0
	if monTracker != nil {
		decisionID := fmt.Sprintf("TS-%d", dp.Timestamp)
		monTracker.RecordDecision(decisionID, value, latencyNS, zScore)
		price = monTracker.CalculatePrice(latencyNS, zScore)
	}

	// Prepare response (Protocol δ-EgressGuard)
	response := Response{
		IsAnomaly:    isAnomaly,
		ZScore:       zScore,
		Timestamp:    dp.Timestamp,
		Value:        dp.Value,
		ProcessingNS: latencyNS,
		Price:        price,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	log.Printf("Processed: TS=%d, Value=%.2f, Anomaly=%t, ZScore=%.3f, Latency=%dns",
		dp.Timestamp, dp.Value, isAnomaly, zScore, latencyNS)
}

// writeErrorResponse writes a standardized error response.
func writeErrorResponse(w http.ResponseWriter, statusCode int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	errorResp := ErrorResponse{
		Error:   code,
		Message: message,
	}
	json.NewEncoder(w).Encode(errorResp)
}

// getClientIP extracts the client IP address from the request.
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP in the chain
		if ip := net.ParseIP(xff); ip != nil {
			return ip.String()
		}
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		if ip := net.ParseIP(xri); ip != nil {
			return ip.String()
		}
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// getDetectorStats returns current detector statistics.
func getDetectorStats() map[string]interface{} {
	if detector == nil {
		return nil
	}
	count, mean, stdDev := detector.GetStats()
	return map[string]interface{}{
		"window_size": detector.WindowSize,
		"threshold":   detector.Threshold,
		"count":       count,
		"mean":        mean,
		"std_dev":     stdDev,
	}
}

// getRateLimitStats returns current rate limiter statistics.
func getRateLimitStats() map[string]interface{} {
	if rateLimit == nil {
		return nil
	}
	return rateLimit.GetStats()
}

// getMonetizationStats returns current monetization statistics.
func getMonetizationStats() map[string]interface{} {
	if monTracker == nil {
		return nil
	}
	return monTracker.GetStats()
}

// setupGracefulShutdown handles graceful shutdown on SIGTERM/SIGINT.
func setupGracefulShutdown(server *chi.Mux) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		log.Println("Shutting down server...")

		// Save final monetization data if enabled
		if monTracker != nil {
			log.Printf("Final monetization stats: %+v", monTracker.GetStats())
		}

		log.Println("Server gracefully stopped")
		os.Exit(0)
	}()
}

// startTime tracks when the server started.
var startTime = time.Now()