package hypervisor

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"internal/blueteam"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// SBOHMetrics represents the Software Bill of Health metrics (Protocol ζ-Hypervisor).
type SBOHMetrics struct {
	Timestamp           time.Time `json:"timestamp"`
	P95LatencyMS        float64   `json:"p95_latency_ms"`
	DecisionSuccessRate float64   `json:"decision_success_rate"`
	MonetizationAccuracy float64  `json:"monetization_accuracy"`
	TotalDecisions      int64     `json:"total_decisions"`
	SuccessfulDecisions int64     `json:"successful_decisions"`
	TotalRevenue        float64   `json:"total_revenue"`
	UptimeSeconds       float64   `json:"uptime_seconds"`
}

// Hypervisor manages the Software Bill of Health (SBOH) for Protocol ζ compliance.
type Hypervisor struct {
	mu                  sync.RWMutex
	metrics             SBOHMetrics
	latencySamples      []float64
	decisionOutcomes    []bool
	revenueTracking     []float64
	startTime           time.Time
	maxSamples          int
}

// Config holds hypervisor configuration.
type Config struct {
	MaxSamples int `json:"max_samples"`
}

// NewHypervisor creates a new hypervisor instance.
func NewHypervisor(config Config) *Hypervisor {
	maxSamples := config.MaxSamples
	if maxSamples <= 0 {
		maxSamples = 10000 // Default to 10k samples
	}

	return &Hypervisor{
		metrics: SBOHMetrics{
			Timestamp: time.Now(),
		},
		latencySamples:   make([]float64, 0, maxSamples),
		decisionOutcomes: make([]bool, 0, maxSamples),
		revenueTracking:  make([]float64, 0, maxSamples),
		startTime:        time.Now(),
		maxSamples:       maxSamples,
	}
}

// RecordDecision records a decision outcome for SBOH tracking.
func (h *Hypervisor) RecordDecision(latencyMS float64, success bool, revenue float64) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Add to rolling samples
	h.latencySamples = append(h.latencySamples, latencyMS)
	h.decisionOutcomes = append(h.decisionOutcomes, success)
	h.revenueTracking = append(h.revenueTracking, revenue)

	// Maintain max samples limit
	if len(h.latencySamples) > h.maxSamples {
		h.latencySamples = h.latencySamples[1:]
		h.decisionOutcomes = h.decisionOutcomes[1:]
		h.revenueTracking = h.revenueTracking[1:]
	}

	// Update metrics
	h.updateMetrics()
}

// updateMetrics recalculates all SBOH metrics.
func (h *Hypervisor) updateMetrics() {
	// Calculate P95 latency
	h.metrics.P95LatencyMS = h.calculateP95Latency()

	// Calculate decision success rate
	h.metrics.DecisionSuccessRate = h.calculateSuccessRate()

	// Calculate monetization accuracy (100% if all decisions are logged)
	h.metrics.MonetizationAccuracy = h.calculateMonetizationAccuracy()

	// Update counters
	h.metrics.TotalDecisions = int64(len(h.decisionOutcomes))
	h.metrics.SuccessfulDecisions = int64(h.countSuccessfulDecisions())
	h.metrics.TotalRevenue = h.sumRevenue()
	h.metrics.UptimeSeconds = time.Since(h.startTime).Seconds()
	h.metrics.Timestamp = time.Now()
}

// calculateP95Latency calculates the 95th percentile latency.
func (h *Hypervisor) calculateP95Latency() float64 {
	if len(h.latencySamples) == 0 {
		return 0.0
	}

	samples := make([]float64, len(h.latencySamples))
	copy(samples, h.latencySamples)

	// Sort samples (simple bubble sort for small arrays)
	for i := 0; i < len(samples); i++ {
		for j := 0; j < len(samples)-1-i; j++ {
			if samples[j] > samples[j+1] {
				samples[j], samples[j+1] = samples[j+1], samples[j]
			}
		}
	}

	p95Index := int(float64(len(samples)) * 0.95)
	if p95Index >= len(samples) {
		p95Index = len(samples) - 1
	}

	return samples[p95Index]
}

// calculateSuccessRate calculates the percentage of successful decisions.
func (h *Hypervisor) calculateSuccessRate() float64 {
	if len(h.decisionOutcomes) == 0 {
		return 100.0
	}

	successful := h.countSuccessfulDecisions()
	return (float64(successful) / float64(len(h.decisionOutcomes))) * 100.0
}

// calculateMonetizationAccuracy calculates monetization logging accuracy.
func (h *Hypervisor) calculateMonetizationAccuracy() float64 {
	// In a perfect system, this should always be 100%
	// Any failure to log revenue would indicate a system fault
	if len(h.revenueTracking) == 0 {
		return 100.0
	}

	// Check for any zero revenue entries that should have been logged
	zeroCount := 0
	for _, revenue := range h.revenueTracking {
		if revenue == 0.0 {
			zeroCount++
		}
	}

	if zeroCount == 0 {
		return 100.0
	}

	return (float64(len(h.revenueTracking)-zeroCount) / float64(len(h.revenueTracking))) * 100.0
}

// countSuccessfulDecisions counts successful decision outcomes.
func (h *Hypervisor) countSuccessfulDecisions() int {
	count := 0
	for _, success := range h.decisionOutcomes {
		if success {
			count++
		}
	}
	return count
}

// sumRevenue calculates total revenue from all decisions.
func (h *Hypervisor) sumRevenue() float64 {
	total := 0.0
	for _, revenue := range h.revenueTracking {
		total += revenue
	}
	return total
}

// GetSBOHMetrics returns the current SBOH metrics.
func (h *Hypervisor) GetSBOHMetrics() SBOHMetrics {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return h.metrics
}

// IsAxiomA2Compliant checks if P95 latency meets Axiom A-2 requirement (≤ 50ms).
func (h *Hypervisor) IsAxiomA2Compliant() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return h.metrics.P95LatencyMS <= 50.0
}

// IsAxiomA4Compliant checks if monetization accuracy is 100%.
func (h *Hypervisor) IsAxiomA4Compliant() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return h.metrics.MonetizationAccuracy >= 99.999 // Allow for floating point precision
}

// GenerateSBOHReport generates a comprehensive SBOH report.
func (h *Hypervisor) GenerateSBOHReport() map[string]interface{} {
	metrics := h.GetSBOHMetrics()

	report := map[string]interface{}{
		"timestamp":             metrics.Timestamp,
		"p95_latency_ms":        metrics.P95LatencyMS,
		"decision_success_rate": metrics.DecisionSuccessRate,
		"monetization_accuracy": metrics.MonetizationAccuracy,
		"total_decisions":       metrics.TotalDecisions,
		"successful_decisions":  metrics.SuccessfulDecisions,
		"total_revenue":         metrics.TotalRevenue,
		"uptime_seconds":        metrics.UptimeSeconds,
		"axiom_a2_compliant":    h.IsAxiomA2Compliant(),
		"axiom_a4_compliant":    h.IsAxiomA4Compliant(),
		"sample_count":          len(h.latencySamples),
	}

	// Log compliance status
	if !h.IsAxiomA2Compliant() {
		log.Printf("WARNING: Axiom A-2 violation - P95 latency %.2fms exceeds 50ms threshold",
			metrics.P95LatencyMS)
	}

	if !h.IsAxiomA4Compliant() {
		log.Printf("WARNING: Axiom A-4 violation - Monetization accuracy %.4f%% below 99.999%% threshold",
			metrics.MonetizationAccuracy)
	}

	return report
}

// ExportSBOH exports SBOH metrics as JSON.
func (h *Hypervisor) ExportSBOH() ([]byte, error) {
	report := h.GenerateSBOHReport()
	return json.MarshalIndent(report, "", "  ")
}

// ObserveExecution wraps function execution with latency tracking for Axiom A-2 compliance.
func (h *Hypervisor) ObserveExecution(fn func() (bool, float64, error)) (bool, float64, error) {
	start := time.Now()
	isAnomaly, zScore, err := fn()
	latency := time.Since(start)

	// Record metrics for SBOH tracking
	latencyMS := float64(latency.Nanoseconds()) / 1e6
	success := err == nil
	price := 0.001 // Default price, would be calculated based on complexity

	h.RecordDecision(latencyMS, success, price)

	return isAnomaly, zScore, err
}

// DefaultConfig returns default hypervisor configuration.
func DefaultConfig() Config {
	return Config{
		MaxSamples: 10000,
	}
}

var (
	// New metric to track A-3 compliance
	TimeToHealDuration = promauto.NewSummary(prometheus.SummaryOpts{
		Name: "radm_time_to_heal_seconds",
		Help: "Duration of a SCGO healing cycle from fault detection to stable state.",
	})
	HealingWindowExceeded = promauto.NewCounter(prometheus.CounterOpts{
		Name: "radm_healing_window_exceeded_total",
		Help: "Total count of times the 60 second healing window was exceeded (CRITICAL AXIOM VIOLATION).",
	})
)

// TriggerHealing is called by the Red Team on fault detection.
func TriggerHealing(healer *blueteam.Healer, faultReason string, isCritical bool) {
	const healingLimit = 60 * time.Second

	var duration time.Duration

	if isCritical {
		duration = healer.ExecuteHardReversion(faultReason)
	} else {
		// Example: initiate soft patch to adjust threshold slightly
		duration = healer.ExecuteSoftPatch(faultReason, 3.2)
	}

	// A-3 Compliance Check
	if duration > healingLimit {
		// CRITICAL FAILURE: The system could not self-correct fast enough.
		log.Fatalf("[SCGO A-3 FAILURE] Time-to-Heal exceeded %s limit! Duration: %s", healingLimit, duration)
		HealingWindowExceeded.Inc()
	}

	// Log the successful Time-to-Heal for SBOH
	TimeToHealDuration.Observe(duration.Seconds())
	log.Printf("[SCGO A-3 SUCCESS] Healing complete and compliant. Duration: %.2fs", duration.Seconds())
}