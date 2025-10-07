package anomaly

import (
	"math"
	"testing"
	"time"
)

// TestAnomalyDetector_BasicFunctionality tests basic anomaly detection
func TestAnomalyDetector_BasicFunctionality(t *testing.T) {
	detector := NewDetector(100, 2.0)

	// Add normal data points
	for i := 0; i < 50; i++ {
		dp := DataPoint{
			Timestamp: int64(1609459200 + i),
			Value:     float64(i),
		}
		isAnomaly, zScore, err := detector.ProcessData(dp)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if isAnomaly {
			t.Errorf("Expected no anomaly for normal data, got anomaly with z-score: %.3f", zScore)
		}
	}

	// Add an anomalous data point
	anomalousDP := DataPoint{
		Timestamp: 1609459200 + 50,
		Value:     1000.0, // Far from normal range
	}
	isAnomaly, zScore, err := detector.ProcessData(anomalousDP)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !isAnomaly {
		t.Errorf("Expected anomaly for value 1000.0, but got z-score: %.3f", zScore)
	}
	if zScore <= 2.0 {
		t.Errorf("Expected high z-score for anomaly, got: %.3f", zScore)
	}
}

// TestAnomalyDetector_WindowSize tests window size constraints
func TestAnomalyDetector_WindowSize(t *testing.T) {
	windowSize := 10
	detector := NewDetector(windowSize, 2.0)

	// Fill the window
	for i := 0; i < windowSize; i++ {
		dp := DataPoint{
			Timestamp: int64(1609459200 + i),
			Value:     1.0,
		}
		detector.ProcessData(dp)
	}

	// Check window size
	count, _, _ := detector.GetStats()
	if count != windowSize {
		t.Errorf("Expected window size %d, got %d", windowSize, count)
	}

	// Add one more point (should evict oldest)
	dp := DataPoint{
		Timestamp: int64(1609459200 + windowSize),
		Value:     2.0,
	}
	detector.ProcessData(dp)

	count, _, _ = detector.GetStats()
	if count != windowSize {
		t.Errorf("Expected window size %d after adding new point, got %d", windowSize, count)
	}
}

// TestAnomalyDetector_ZeroVariance tests zero variance scenario
func TestAnomalyDetector_ZeroVariance(t *testing.T) {
	detector := NewDetector(10, 2.0)

	// Add identical data points
	for i := 0; i < 10; i++ {
		dp := DataPoint{
			Timestamp: int64(1609459200 + i),
			Value:     5.0,
		}
		detector.ProcessData(dp)
	}

	// Add a different point
	dp := DataPoint{
		Timestamp: 1609459200 + 10,
		Value:     6.0,
	}
	isAnomaly, zScore, err := detector.ProcessData(dp)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !isAnomaly {
		t.Errorf("Expected anomaly for different value in zero variance window, got z-score: %.3f", zScore)
	}
}

// TestAnomalyDetector_InsufficientData tests behavior with insufficient data
func TestAnomalyDetector_InsufficientData(t *testing.T) {
	detector := NewDetector(10, 2.0)

	// Add only one data point
	dp := DataPoint{
		Timestamp: 1609459200,
		Value:     5.0,
	}
	isAnomaly, zScore, err := detector.ProcessData(dp)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if isAnomaly {
		t.Errorf("Expected no anomaly for insufficient data, got z-score: %.3f", zScore)
	}
	if zScore != 0.0 {
		t.Errorf("Expected z-score 0.0 for insufficient data, got: %.3f", zScore)
	}
}

// TestAnomalyDetector_ThreadSafety tests concurrent access
func TestAnomalyDetector_ThreadSafety(t *testing.T) {
	detector := NewDetector(100, 2.0)

	// Run concurrent operations
	done := make(chan bool, 2)

	go func() {
		for i := 0; i < 100; i++ {
			dp := DataPoint{
				Timestamp: int64(1609459200 + i),
				Value:     float64(i),
			}
			detector.ProcessData(dp)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			_, _, _ = detector.GetStats()
			time.Sleep(time.Microsecond)
		}
		done <- true
	}()

	// Wait for both goroutines
	<-done
	<-done
}

// TestAnomalyDetector_Reset tests the reset functionality
func TestAnomalyDetector_Reset(t *testing.T) {
	detector := NewDetector(10, 2.0)

	// Add some data
	for i := 0; i < 5; i++ {
		dp := DataPoint{
			Timestamp: int64(1609459200 + i),
			Value:     float64(i),
		}
		detector.ProcessData(dp)
	}

	// Verify data exists
	count, _, _ := detector.GetStats()
	if count != 5 {
		t.Errorf("Expected 5 data points before reset, got %d", count)
	}

	// Reset
	detector.Reset()

	// Verify data is cleared
	count, _, _ = detector.GetStats()
	if count != 0 {
		t.Errorf("Expected 0 data points after reset, got %d", count)
	}
}

// TestAnomalyDetector_EdgeCases tests various edge cases
func TestAnomalyDetector_EdgeCases(t *testing.T) {
	testCases := []struct {
		name      string
		windowSize int
		threshold float64
		data      []DataPoint
		expectAnomaly bool
	}{
		{
			name:      "Very small window",
			windowSize: 2,
			threshold: 1.0,
			data: []DataPoint{
				{Timestamp: 1609459200, Value: 1.0},
				{Timestamp: 1609459201, Value: 100.0},
			},
			expectAnomaly: true,
		},
		{
			name:      "Very high threshold",
			windowSize: 10,
			threshold: 10.0,
			data: []DataPoint{
				{Timestamp: 1609459200, Value: 1.0},
				{Timestamp: 1609459201, Value: 2.0},
				{Timestamp: 1609459202, Value: 3.0},
				{Timestamp: 1609459203, Value: 100.0},
			},
			expectAnomaly: false,
		},
		{
			name:      "Negative values",
			windowSize: 5,
			threshold: 2.0,
			data: []DataPoint{
				{Timestamp: 1609459200, Value: -10.0},
				{Timestamp: 1609459201, Value: -11.0},
				{Timestamp: 1609459202, Value: -12.0},
				{Timestamp: 1609459203, Value: -100.0},
			},
			expectAnomaly: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			detector := NewDetector(tc.windowSize, tc.threshold)

			// Add initial data points
			for i := 0; i < len(tc.data)-1; i++ {
				detector.ProcessData(tc.data[i])
			}

			// Test the final data point
			isAnomaly, _, err := detector.ProcessData(tc.data[len(tc.data)-1])
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if isAnomaly != tc.expectAnomaly {
				t.Errorf("Expected anomaly: %v, got: %v", tc.expectAnomaly, isAnomaly)
			}
		})
	}
}

// BenchmarkAnomalyDetector_ProcessData benchmarks the ProcessData method
func BenchmarkAnomalyDetector_ProcessData(b *testing.B) {
	detector := NewDetector(1000, 3.0)

	// Pre-fill with some data
	for i := 0; i < 500; i++ {
		dp := DataPoint{
			Timestamp: int64(1609459200 + i),
			Value:     float64(i),
		}
		detector.ProcessData(dp)
	}

	dp := DataPoint{
		Timestamp: 1609459200 + 1000,
		Value:     1000.0,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		detector.ProcessData(dp)
	}
}

// BenchmarkAnomalyDetector_GetStats benchmarks the GetStats method
func BenchmarkAnomalyDetector_GetStats(b *testing.B) {
	detector := NewDetector(1000, 3.0)

	// Pre-fill with data
	for i := 0; i < 1000; i++ {
		dp := DataPoint{
			Timestamp: int64(1609459200 + i),
			Value:     float64(i),
		}
		detector.ProcessData(dp)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		detector.GetStats()
	}
}