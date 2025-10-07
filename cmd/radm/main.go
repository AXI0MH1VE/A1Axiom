package main

import (
	"crypto/sha256"
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
	"internal/audit"
	"internal/blueteam"
	"internal/config"
	"internal/hypervisor"
	"internal/monetization"
	"internal/ratelimit"
	"internal/redteam"
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
	detector    *anomaly.AnomalyDetector
	monTracker  *monetization.MonetizationTracker
	validator   *validation.DataPointValidator
	rateLimit   *ratelimit.RateLimiter
	hypervisor  *hypervisor.Hypervisor
	redTeam     *redteam.RedTeam
	blueTeam    *blueteam.BlueTeam
	healer      *blueteam.Healer
	auditor     *audit.Auditor
	cfg         *config.Config
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

	// Initialize hypervisor (Protocol ζ-Hypervisor)
	hypConfig := hypervisor.DefaultConfig()
	hypervisorInstance := hypervisor.NewHypervisor(hypConfig)

	// Initialize Red Team (Protocol β-RedTeam)
	redTeamInstance := redteam.NewRedTeam()
	redTeamInstance.SetupDefaultFaults()
	redTeamInstance.StartFaultCleanupRoutine()

	// Initialize Auditor for comprehensive compliance verification
	auditConfig := audit.DefaultConfig()
	auditorInstance, err := audit.NewAuditor(auditConfig)
	if err != nil {
		log.Fatalf("Failed to initialize auditor: %v", err)
	}

	// Initialize Blue Team for self-healing mechanisms (Protocol β-RedTeam/Blue Team)
	blueTeamConfig := blueteam.DefaultConfig()
	blueTeamInstance := blueteam.NewBlueTeam(blueTeamConfig)
	blueTeamInstance.StartMonitoring()

	// Initialize Blue Team Healer
	healerInstance := blueteam.NewHealer(detector)
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

	// System endpoints (All Protocols)
	r.Get("/metrics", metricsHandler)
	r.Get("/sboh", sbohHandler)
	r.Get("/redteam/status", redTeamStatusHandler)
	r.Post("/redteam/fault/{type}", redTeamFaultHandler)
	r.Get("/blueteam/status", blueTeamStatusHandler)
	r.Post("/blueteam/heal/{type}", blueTeamHealHandler)
	r.Get("/audit/events", auditEventsHandler)
	r.Get("/audit/compliance", auditComplianceHandler)

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
				// Audit rate limit violation
				if auditorInstance != nil {
					auditorInstance.LogRateLimit(false, getClientIP(r), middleware.GetReqID(r.Context()))
				}
				log.Printf("Rate limit exceeded for IP: %s", getClientIP(r))
				writeErrorResponse(w, http.StatusTooManyRequests, "RATE_LIMIT_EXCEEDED",
					"Rate limit exceeded. Please try again later.")
				return
			}
			// Audit successful rate limit check
			if auditorInstance != nil {
				auditorInstance.LogRateLimit(true, getClientIP(r), middleware.GetReqID(r.Context()))
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
		"detector_stats":     getDetectorStats(),
		"rate_limit_stats":   getRateLimitStats(),
		"monetization_stats": getMonetizationStats(),
		"sboh_summary":       getSBOHSummary(),
		"redteam_stats":      getRedTeamStats(),
		"uptime_seconds":     time.Since(startTime).Seconds(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// sbohHandler provides comprehensive SBOH metrics (Protocol ζ-Hypervisor).
func sbohHandler(w http.ResponseWriter, r *http.Request) {
	if hypervisorInstance == nil {
		writeErrorResponse(w, http.StatusServiceUnavailable, "HYPERVISOR_UNAVAILABLE",
			"Hypervisor not initialized")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(hypervisorInstance.GenerateSBOHReport())
}

// ingestHandler handles data ingestion requests.
func ingestHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Parse request body
	var dp anomaly.DataPoint
	if err := json.NewDecoder(r.Body).Decode(&dp); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "INVALID_JSON",
			"Invalid JSON in request body")
		return
	}

	// 1. Input Validation (Now using validator/v10 via the module)
	if err := validation.ValidateDataPoint(dp); err != nil {
		// Example of triggering a Soft Patch on persistent validation failures
		go hypervisor.TriggerHealing(healerInstance, "High validation failure rate detected", false)
		writeErrorResponse(w, http.StatusBadRequest, "VALIDATION_FAILED", fmt.Sprintf("Schema Validation Failure: %v", err))
		return
	}

	// 2. Process Data (Wrapped by Hypervisor for A-2 latency tracking)
	// The closure passed to ObserveExecution calls the core logic.
	isAnomaly, zScore, err := hypervisorInstance.ObserveExecution(func() (bool, float64, error) {
		// Inject processing faults (Protocol β-RedTeam)
		if redTeamInstance != nil {
			if err := redTeamInstance.InjectProcessingFault(); err != nil {
				// Audit fault injection
				if auditorInstance != nil {
					auditorInstance.LogFaultInjection("processing", true, time.Second*30)
				}
				return false, 0.0, err
			}
		}

		return detector.ProcessData(dp)
	})

	if err != nil {
		// Example of triggering a Hard Reversion on critical error
		go hypervisor.TriggerHealing(healerInstance, fmt.Sprintf("Critical algorithm error: %v", err), true)
		writeErrorResponse(w, http.StatusInternalServerError, "PROCESSING_ERROR",
			"Internal processing error")
		return
	}

	// Audit decision
	if auditorInstance != nil {
		decisionID := fmt.Sprintf("TS-%d", dp.Timestamp)
		auditorInstance.LogDecision(decisionID, isAnomaly, zScore, latencyNS, getClientIP(r))
	}

	// Create output hash for determinism verification
	outputData := map[string]interface{}{
		"is_anomaly":   isAnomaly,
		"z_score":      zScore,
		"timestamp":    dp.Timestamp,
		"value":        dp.Value,
		"processing_ns": time.Since(start).Nanoseconds(),
	}
	outputBytes, _ := json.Marshal(outputData)
	outputHash := fmt.Sprintf("%x", sha256.Sum256(outputBytes))

	// Verify determinism (Axiom A-1)
	if err := detector.VerifyDeterminism(inputHash, outputHash); err != nil {
		log.Printf("Determinism violation detected: %v", err)
		// Create checkpoint for recovery
		checkpoint := detector.CreateCheckpoint(inputHash, outputHash)
		log.Printf("Created recovery checkpoint: %s", checkpoint.StateHash[:16]+"...")
	}

	// Record monetization data (Axiom A-4 Hook)
	latencyNS := time.Since(start).Nanoseconds()

	// Inject latency faults (Protocol β-RedTeam)
	originalLatency := time.Duration(latencyNS)
	if redTeamInstance != nil {
		injectedLatency := redTeamInstance.InjectLatency(originalLatency)
		if injectedLatency != originalLatency {
			// Audit latency fault injection
			if auditorInstance != nil {
				auditorInstance.LogFaultInjection("latency", true, time.Minute*2)
			}
		}
		latencyNS = injectedLatency.Nanoseconds()
	}

	latencyMS := float64(latencyNS) / 1e6
	price := 0.0
	success := err == nil

	if monTracker != nil {
		decisionID := fmt.Sprintf("TS-%d", dp.Timestamp)
		monTracker.RecordDecision(decisionID, value, latencyNS, zScore)
		price = monTracker.CalculatePrice(latencyNS, zScore)
	}

	// Record in hypervisor for SBOH tracking (Protocol ζ-Hypervisor)
	if hypervisorInstance != nil {
		hypervisorInstance.RecordDecision(latencyMS, success, price)

		// Self-healing: Check for compliance violations and trigger healing
		if blueTeamInstance != nil {
			// Check Axiom A-2 compliance (P95 latency ≤ 50ms)
			if !hypervisorInstance.IsAxiomA2Compliant() {
				// Audit compliance failure
				if auditorInstance != nil {
					auditorInstance.LogCompliance("γ-Axiomatic Control", "A-2",
						false, map[string]interface{}{"p95_latency_ms": latencyMS})
				}
				// Trigger self-healing
				blueTeamInstance.HealOnDemand(blueteam.IssueHighLatency, blueteam.StrategyCircuitBreaker)
			} else {
				// Audit compliance success
				if auditorInstance != nil {
					auditorInstance.LogCompliance("γ-Axiomatic Control", "A-2",
						true, map[string]interface{}{"p95_latency_ms": latencyMS})
				}
			}

			// Check Axiom A-4 compliance (monetization accuracy)
			if !hypervisorInstance.IsAxiomA4Compliant() {
				// Audit compliance failure
				if auditorInstance != nil {
					auditorInstance.LogCompliance("ζ-Hypervisor", "A-4",
						false, map[string]interface{}{"monetization_accuracy": price})
				}
				// Trigger self-healing
				blueTeamInstance.HealOnDemand(blueteam.IssueComplianceFailure, blueteam.StrategyConfigReload)
			} else {
				// Audit compliance success
				if auditorInstance != nil {
					auditorInstance.LogCompliance("ζ-Hypervisor", "A-4",
						true, map[string]interface{}{"monetization_accuracy": price})
				}
			}
		}
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

// getSBOHSummary returns a summary of SBOH metrics for the main metrics endpoint.
func getSBOHSummary() map[string]interface{} {
	if hypervisorInstance == nil {
		return nil
	}

	metrics := hypervisorInstance.GetSBOHMetrics()
	return map[string]interface{}{
		"p95_latency_ms":        metrics.P95LatencyMS,
		"decision_success_rate": metrics.DecisionSuccessRate,
		"monetization_accuracy": metrics.MonetizationAccuracy,
		"axiom_a2_compliant":    hypervisorInstance.IsAxiomA2Compliant(),
		"axiom_a4_compliant":    hypervisorInstance.IsAxiomA4Compliant(),
		"total_decisions":       metrics.TotalDecisions,
		"total_revenue":         metrics.TotalRevenue,
	}
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

		// Stop Blue Team monitoring
		if blueTeamInstance != nil {
			blueTeamInstance.StopMonitoring()
			log.Println("Blue Team monitoring stopped")
		}

		// Close healer (no special cleanup needed)
		if healerInstance != nil {
			log.Println("Blue Team healer shutdown complete")
		}

		// Close auditor
		if auditorInstance != nil {
			if err := auditorInstance.Close(); err != nil {
				log.Printf("Error closing auditor: %v", err)
			} else {
				log.Println("Auditor closed successfully")
			}
		}

		log.Println("Server gracefully stopped")
		os.Exit(0)
	}()
}

// startTime tracks when the server started.
var startTime = time.Now()

// redTeamStatusHandler provides Red Team status and statistics.
func redTeamStatusHandler(w http.ResponseWriter, r *http.Request) {
	if redTeamInstance == nil {
		writeErrorResponse(w, http.StatusServiceUnavailable, "REDTEAM_UNAVAILABLE",
			"Red Team not initialized")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(redTeamInstance.GetFaultStats())
}

// redTeamFaultHandler allows manual control of fault injection.
func redTeamFaultHandler(w http.ResponseWriter, r *http.Request) {
	if redTeamInstance == nil {
		writeErrorResponse(w, http.StatusServiceUnavailable, "REDTEAM_UNAVAILABLE",
			"Red Team not initialized")
		return
	}

	faultType := chi.URLParam(r, "type")
	if faultType == "" {
		writeErrorResponse(w, http.StatusBadRequest, "MISSING_FAULT_TYPE",
			"Fault type must be specified")
		return
	}

	action := r.URL.Query().Get("action")
	if action == "" {
		action = "toggle"
	}

	switch redteam.FaultType(faultType) {
	case redteam.FaultLatency, redteam.FaultValidationFail, redteam.FaultProcessingFail:
		switch action {
		case "enable":
			redTeamInstance.EnableFault(redteam.FaultType(faultType))
		case "disable":
			redTeamInstance.DisableFault(redteam.FaultType(faultType))
		default:
			// Toggle behavior
			activeFaults := redTeamInstance.GetActiveFaults()
			if _, isActive := activeFaults[redteam.FaultType(faultType)]; isActive {
				redTeamInstance.DisableFault(redteam.FaultType(faultType))
			} else {
				redTeamInstance.EnableFault(redteam.FaultType(faultType))
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"fault_type": faultType,
			"action":     action,
			"status":     "success",
		})

	default:
		writeErrorResponse(w, http.StatusBadRequest, "INVALID_FAULT_TYPE",
			"Unsupported fault type: "+faultType)
	}
}

// getRedTeamStats returns current Red Team statistics.
func getRedTeamStats() map[string]interface{} {
	if redTeamInstance == nil {
		return nil
	}
	return redTeamInstance.GetFaultStats()
}

// auditEventsHandler provides recent audit events.
func auditEventsHandler(w http.ResponseWriter, r *http.Request) {
	if auditorInstance == nil {
		writeErrorResponse(w, http.StatusServiceUnavailable, "AUDITOR_UNAVAILABLE",
			"Auditor not initialized")
		return
	}

	// Get limit from query parameter
	limit := 100 // default
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsedLimit := parseInt(limitStr); parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	events := auditorInstance.GetEvents(limit)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"events": events,
		"count":  len(events),
		"limit":  limit,
	})
}

// auditComplianceHandler provides compliance reports.
func auditComplianceHandler(w http.ResponseWriter, r *http.Request) {
	if auditorInstance == nil {
		writeErrorResponse(w, http.StatusServiceUnavailable, "AUDITOR_UNAVAILABLE",
			"Auditor not initialized")
		return
	}

	// Get time range from query parameters
	since := time.Now().Add(-time.Hour) // default to last hour
	if sinceStr := r.URL.Query().Get("since"); sinceStr != "" {
		if parsedTime, err := time.Parse(time.RFC3339, sinceStr); err == nil {
			since = parsedTime
		}
	}

	report := auditorInstance.GetComplianceReport(since)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}

// Helper function to parse integers safely
func parseInt(s string) int {
	// Simple implementation - in production, use strconv.Atoi with error handling
	result := 0
	for _, char := range s {
		if char >= '0' && char <= '9' {
			result = result*10 + int(char-'0')
		} else {
			return 0
		}
	}
	return result
}

// blueTeamStatusHandler provides Blue Team status and healing history.
func blueTeamStatusHandler(w http.ResponseWriter, r *http.Request) {
	if blueTeamInstance == nil {
		writeErrorResponse(w, http.StatusServiceUnavailable, "BLUETEAM_UNAVAILABLE",
			"Blue Team not initialized")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(blueTeamInstance.GetHealingStats())
}

// blueTeamHealHandler allows manual triggering of healing actions.
func blueTeamHealHandler(w http.ResponseWriter, r *http.Request) {
	if blueTeamInstance == nil {
		writeErrorResponse(w, http.StatusServiceUnavailable, "BLUETEAM_UNAVAILABLE",
			"Blue Team not initialized")
		return
	}

	issueType := chi.URLParam(r, "type")
	if issueType == "" {
		writeErrorResponse(w, http.StatusBadRequest, "MISSING_ISSUE_TYPE",
			"Issue type must be specified")
		return
	}

	strategy := r.URL.Query().Get("strategy")
	if strategy == "" {
		strategy = "circuit_breaker" // default strategy
	}

	// Map string to enum values
	var issue blueteam.IssueType
	var healStrategy blueteam.HealingStrategy

	switch issueType {
	case "high_latency":
		issue = blueteam.IssueHighLatency
	case "high_error_rate":
		issue = blueteam.IssueHighErrorRate
	case "resource_exhaustion":
		issue = blueteam.IssueResourceExhaustion
	case "compliance_failure":
		issue = blueteam.IssueComplianceFailure
	default:
		writeErrorResponse(w, http.StatusBadRequest, "INVALID_ISSUE_TYPE",
			"Unsupported issue type: "+issueType)
		return
	}

	switch strategy {
	case "reset_detector":
		healStrategy = blueteam.StrategyResetDetector
	case "circuit_breaker":
		healStrategy = blueteam.StrategyCircuitBreaker
	case "fallback_mode":
		healStrategy = blueteam.StrategyFallbackMode
	case "resource_cleanup":
		healStrategy = blueteam.StrategyResourceCleanup
	case "config_reload":
		healStrategy = blueteam.StrategyConfigReload
	default:
		writeErrorResponse(w, http.StatusBadRequest, "INVALID_STRATEGY",
			"Unsupported healing strategy: "+strategy)
		return
	}

	// Trigger healing
	action := blueTeamInstance.HealOnDemand(issue, healStrategy)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"healing_action": action,
		"status":         "success",
	})
}

// getBlueTeamStats returns current Blue Team statistics.
func getBlueTeamStats() map[string]interface{} {
	if blueTeamInstance == nil {
		return nil
	}
	return blueTeamInstance.GetHealingStats()
}