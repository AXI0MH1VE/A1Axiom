package anomaly

import (
	"fmt"
	"math"
	"sync"
)

// AnomalyDetector holds the state and configuration for the anomaly detection logic.
type AnomalyDetector struct {
	mu           sync.RWMutex
	WindowSize   int
	Threshold    float64
	dataWindow   []float64
	sum          float64
	sumOfSquares float64
}

// DataPoint represents a single ingestion event.
type DataPoint struct {
	Timestamp int64   `json:"timestamp" validate:"required,gt=0"`
	Value     float64 `json:"value" validate:"required"`
}

// NewDetector initializes a new AnomalyDetector.
func NewDetector(windowSize int, threshold float64) *AnomalyDetector {
	return &AnomalyDetector{
		WindowSize: windowSize,
		Threshold:  threshold,
		dataWindow: make([]float64, 0, windowSize),
	}
}

// ProcessData ingests a new data point, updates the window, and checks for an anomaly.
// Big O Notation: O(1) amortized.
func (ad *AnomalyDetector) ProcessData(dp DataPoint) (isAnomaly bool, zScore float64, err error) {
	ad.mu.Lock()
	defer ad.mu.Unlock()

	newValue := dp.Value

	if len(ad.dataWindow) >= ad.WindowSize {
		oldValue := ad.dataWindow[0]
		ad.sum -= oldValue
		ad.sumOfSquares -= oldValue * oldValue
		ad.dataWindow = ad.dataWindow[1:]
	}

	ad.sum += newValue
	ad.sumOfSquares += newValue * newValue
	ad.dataWindow = append(ad.dataWindow, newValue)

	currentSize := len(ad.dataWindow)
	if currentSize < 2 {
		return false, 0.0, nil
	}

	mean := ad.sum / float64(currentSize)
	variance := (ad.sumOfSquares / float64(currentSize)) - (mean * mean)

	if variance < 0 {
		variance = 0
	}

	stdDev := math.Sqrt(variance)

	if stdDev == 0 {
		if newValue != mean {
			return true, math.MaxFloat64, nil
		}
		return false, 0.0, nil
	}

	zScore = math.Abs((newValue - mean) / stdDev)
	isAnomaly = zScore > ad.Threshold

	return isAnomaly, zScore, nil
}

// GetStats returns current statistics about the data window for monitoring purposes.
func (ad *AnomalyDetector) GetStats() (count int, mean float64, stdDev float64) {
	ad.mu.RLock()
	defer ad.mu.RUnlock()

	currentSize := len(ad.dataWindow)
	if currentSize == 0 {
		return 0, 0.0, 0.0
	}

	mean = ad.sum / float64(currentSize)
	variance := (ad.sumOfSquares / float64(currentSize)) - (mean * mean)

	if variance < 0 {
		variance = 0
	}

	stdDev = math.Sqrt(variance)
	return currentSize, mean, stdDev
}

// Reset clears the data window and resets statistics.
func (ad *AnomalyDetector) Reset() {
	ad.mu.Lock()
	defer ad.mu.Unlock()

	ad.dataWindow = make([]float64, 0, ad.WindowSize)
	ad.sum = 0.0
	ad.sumOfSquares = 0.0
}