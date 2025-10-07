package monetization

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

func TestMonetizationTracker_RecordDecision(t *testing.T) {
	// Create temporary file for testing
	tmpFile, err := os.CreateTemp("", "pov_test_*.jsonl")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	config := Config{
		BasePrice:            0.001,
		ComplexityMultiplier: 0.1,
		OutputFile:           tmpFile.Name(),
	}

	tracker := NewTracker(config)

	// Record a decision
	decisionID := "test-decision-1"
	value := 42.5
	processingNS := int64(150000) // 150 microseconds
	zScore := 3.24

	tracker.RecordDecision(decisionID, value, processingNS, zScore)

	// Check stats
	stats := tracker.GetStats()
	if stats["total_decisions"].(int) != 1 {
		t.Errorf("Expected 1 decision, got %d", stats["total_decisions"])
	}

	expectedPrice := tracker.CalculatePrice(processingNS, zScore)
	if stats["total_value"].(float64) != expectedPrice {
		t.Errorf("Expected total value %.6f, got %.6f", expectedPrice, stats["total_value"])
	}
}

func TestMonetizationTracker_CalculatePrice(t *testing.T) {
	config := Config{
		BasePrice:            0.001,
		ComplexityMultiplier: 0.1,
		OutputFile:           "test.jsonl",
	}

	tracker := NewTracker(config)

	testCases := []struct {
		name         string
		processingNS int64
		zScore       float64
		expectedMin  float64
		expectedMax  float64
	}{
		{
			name:         "Normal processing",
			processingNS: 100000, // 100 microseconds
			zScore:       1.0,
			expectedMin:  0.001,
			expectedMax:  0.002,
		},
		{
			name:         "High latency",
			processingNS: 1000000, // 1 millisecond
			zScore:       2.0,
			expectedMin:  0.001,
			expectedMax:  0.003,
		},
		{
			name:         "High z-score",
			processingNS: 100000,
			zScore:       5.0,
			expectedMin:  0.001,
			expectedMax:  0.003,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			price := tracker.CalculatePrice(tc.processingNS, tc.zScore)
			if price < tc.expectedMin || price > tc.expectedMax {
				t.Errorf("Expected price between %.6f and %.6f, got %.6f",
					tc.expectedMin, tc.expectedMax, price)
			}
		})
	}
}

func TestMonetizationTracker_GetAverageLatency(t *testing.T) {
	config := Config{
		BasePrice:            0.001,
		ComplexityMultiplier: 0.1,
		OutputFile:           "test.jsonl",
	}

	tracker := NewTracker(config)

	// Initially no data
	avgLatency := tracker.GetAverageLatency()
	if avgLatency != 0 {
		t.Errorf("Expected average latency 0 for no data, got %d", avgLatency)
	}

	// Record some decisions
	latencies := []int64{100000, 200000, 300000}
	for i, latency := range latencies {
		tracker.RecordDecision(
			fmt.Sprintf("test-%d", i),
			float64(i),
			latency,
			1.0,
		)
	}

	expectedAvg := (100000 + 200000 + 300000) / 3
	avgLatency = tracker.GetAverageLatency()
	if avgLatency != expectedAvg {
		t.Errorf("Expected average latency %d, got %d", expectedAvg, avgLatency)
	}
}

func TestMonetizationTracker_GetAnomalyRate(t *testing.T) {
	config := Config{
		BasePrice:            0.001,
		ComplexityMultiplier: 0.1,
		OutputFile:           "test.jsonl",
	}

	tracker := NewTracker(config)

	// Initially no anomalies
	rate := tracker.GetAnomalyRate()
	if rate != 0.0 {
		t.Errorf("Expected anomaly rate 0.0 for no data, got %.2f", rate)
	}

	// Record decisions with mixed anomaly status
	// Note: Simplified anomaly detection for testing (zScore > 0)
	tracker.RecordDecision("normal-1", 1.0, 100000, 0.5)  // Not anomaly
	tracker.RecordDecision("normal-2", 2.0, 100000, 1.0)  // Not anomaly
	tracker.RecordDecision("anomaly-1", 3.0, 100000, 2.0) // Anomaly
	tracker.RecordDecision("anomaly-2", 4.0, 100000, 3.0) // Anomaly

	rate = tracker.GetAnomalyRate()
	expectedRate := 50.0 // 2 out of 4 decisions are anomalies
	if rate != expectedRate {
		t.Errorf("Expected anomaly rate %.2f, got %.2f", expectedRate, rate)
	}
}

func TestMonetizationTracker_Persistence(t *testing.T) {
	// Create temporary file for testing
	tmpFile, err := os.CreateTemp("", "pov_test_*.jsonl")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	config := Config{
		BasePrice:            0.001,
		ComplexityMultiplier: 0.1,
		OutputFile:           tmpFile.Name(),
	}

	tracker := NewTracker(config)

	// Record a decision
	decisionID := "test-persistence"
	value := 42.5
	processingNS := int64(150000)
	zScore := 3.24

	tracker.RecordDecision(decisionID, value, processingNS, zScore)

	// Give some time for async persistence
	time.Sleep(100 * time.Millisecond)

	// Read the file and verify content
	content, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to read temp file: %v", err)
	}

	if len(content) == 0 {
		t.Error("Expected file to contain data, but it was empty")
	}

	// Parse the JSON line
	var record DecisionRecord
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) != 1 {
		t.Fatalf("Expected 1 line in file, got %d", len(lines))
	}

	err = json.Unmarshal([]byte(lines[0]), &record)
	if err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if record.DecisionID != decisionID {
		t.Errorf("Expected decision ID %s, got %s", decisionID, record.DecisionID)
	}

	if record.Value != value {
		t.Errorf("Expected value %.2f, got %.2f", value, record.Value)
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.BasePrice <= 0 {
		t.Error("Expected positive base price in default config")
	}

	if config.ComplexityMultiplier <= 0 {
		t.Error("Expected positive complexity multiplier in default config")
	}

	if config.OutputFile == "" {
		t.Error("Expected non-empty output file in default config")
	}
}

// Helper variables for testing
var (
	// Use a non-existent file path for testing
	testOutputFile = "/nonexistent/path/test_pov.jsonl"
)