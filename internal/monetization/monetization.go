package monetization

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

// DecisionRecord represents a single decision event for Proof-of-Value (PoV) tracking.
type DecisionRecord struct {
	DecisionID    string    `json:"decision_id"`
	Timestamp     time.Time `json:"timestamp"`
	ProcessingNS  int64     `json:"processing_ns"`
	ZScore        float64   `json:"z_score"`
	Value         float64   `json:"value"`
	IsAnomaly     bool      `json:"is_anomaly"`
}

// MonetizationTracker handles Proof-of-Value (PoV) logging and financial calculations.
type MonetizationTracker struct {
	mu           sync.RWMutex
	records      []DecisionRecord
	basePrice    float64
	complexityMultiplier float64
	outputFile   string
}

// Config holds monetization configuration.
type Config struct {
	BasePrice            float64 `json:"base_price"`
	ComplexityMultiplier float64 `json:"complexity_multiplier"`
	OutputFile           string  `json:"output_file"`
}

// NewTracker creates a new MonetizationTracker with the given configuration.
func NewTracker(config Config) *MonetizationTracker {
	return &MonetizationTracker{
		records:              make([]DecisionRecord, 0),
		basePrice:            config.BasePrice,
		complexityMultiplier: config.ComplexityMultiplier,
		outputFile:           config.OutputFile,
	}
}

// RecordDecision logs a decision event for PoV tracking and financial calculation.
func (mt *MonetizationTracker) RecordDecision(decisionID string, value float64, processingNS int64, zScore float64) {
	mt.mu.Lock()
	defer mt.mu.Unlock()

	record := DecisionRecord{
		DecisionID:   decisionID,
		Timestamp:    time.Now(),
		ProcessingNS: processingNS,
		ZScore:       zScore,
		Value:        value,
		IsAnomaly:    zScore > 0, // Simplified: any z-score > 0 indicates anomaly
	}

	mt.records = append(mt.records, record)

	// Log for immediate feedback
	log.Printf("PoV Event: %s | Latency: %d ns | Z-Score: %.3f | Price: $%.6f",
		decisionID, processingNS, zScore, mt.CalculatePrice(processingNS, zScore))

	// Persist to file asynchronously for performance
	go mt.persistRecord(record)
}

// CalculatePrice computes the dynamic price based on processing complexity and latency.
func (mt *MonetizationTracker) CalculatePrice(processingNS int64, zScore float64) float64 {
	// Base price adjusted by processing time (latency affects pricing)
	latencyFactor := 1.0 + (float64(processingNS) / 1e9) * mt.complexityMultiplier

	// Anomaly detection complexity factor (higher z-score = more complex analysis)
	complexityFactor := 1.0 + (zScore / 10.0) * mt.complexityMultiplier

	return mt.basePrice * latencyFactor * complexityFactor
}

// GetTotalValue calculates the total monetary value of all processed decisions.
func (mt *MonetizationTracker) GetTotalValue() float64 {
	mt.mu.RLock()
	defer mt.mu.RUnlock()

	total := 0.0
	for _, record := range mt.records {
		total += mt.CalculatePrice(record.ProcessingNS, record.ZScore)
	}
	return total
}

// GetAverageLatency returns the average processing latency in nanoseconds.
func (mt *MonetizationTracker) GetAverageLatency() int64 {
	mt.mu.RLock()
	defer mt.mu.RUnlock()

	if len(mt.records) == 0 {
		return 0
	}

	total := int64(0)
	for _, record := range mt.records {
		total += record.ProcessingNS
	}
	return total / int64(len(mt.records))
}

// GetAnomalyRate returns the percentage of decisions that were anomalies.
func (mt *MonetizationTracker) GetAnomalyRate() float64 {
	mt.mu.RLock()
	defer mt.mu.RUnlock()

	if len(mt.records) == 0 {
		return 0.0
	}

	anomalies := 0
	for _, record := range mt.records {
		if record.IsAnomaly {
			anomalies++
		}
	}
	return (float64(anomalies) / float64(len(mt.records))) * 100.0
}

// GetStats returns comprehensive monetization statistics.
func (mt *MonetizationTracker) GetStats() map[string]interface{} {
	mt.mu.RLock()
	defer mt.mu.RUnlock()

	return map[string]interface{}{
		"total_decisions":    len(mt.records),
		"total_value":        mt.GetTotalValue(),
		"average_latency_ns": mt.GetAverageLatency(),
		"anomaly_rate_pct":   mt.GetAnomalyRate(),
		"base_price":         mt.basePrice,
	}
}

// persistRecord writes a single record to the output file.
func (mt *MonetizationTracker) persistRecord(record DecisionRecord) {
	file, err := os.OpenFile(mt.outputFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("Error opening monetization file: %v", err)
		return
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	if err := encoder.Encode(record); err != nil {
		log.Printf("Error writing monetization record: %v", err)
	}
}

// DefaultConfig returns a default monetization configuration.
func DefaultConfig() Config {
	return Config{
		BasePrice:            0.001, // $0.001 base price per decision
		ComplexityMultiplier: 0.1,   // 10% price increase per complexity unit
		OutputFile:           "pov_records.jsonl",
	}
}