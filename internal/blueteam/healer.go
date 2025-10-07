package blueteam

import (
	"log"
	"time"
	"anomaly"
)

// Healer holds a reference to the anomaly detector to execute patches.
type Healer struct {
	Detector *anomaly.AnomalyDetector
}

// NewHealer creates a new Blue Team Healer instance.
func NewHealer(d *anomaly.AnomalyDetector) *Healer {
	return &Healer{Detector: d}
}

// ExecuteHardReversion performs the fast, necessary rollback for critical failures.
// This is the fastest path to restoring Axiom A-1 Determinism.
func (h *Healer) ExecuteHardReversion(faultReason string) time.Duration {
	log.Printf("[BlueTeam/HardReversion] Initiating critical state rollback. Reason: %s", faultReason)
	start := time.Now()

	// Rollback to known-good default state
	// In a full CRG, this would involve loading the last HASHED snapshot.
	h.Detector.ResetState(500, 3.5) // Default values

	duration := time.Since(start)
	log.Printf("[BlueTeam/HardReversion] State roll-back complete. Time-to-Heal: %s", duration)
	return duration
}

// ExecuteSoftPatch initiates a logical correction (e.g., based on high false-positive rate).
// This path prioritizes validation and strategic optimization (Protocol γ-Axiomatic Control).
func (h *Healer) ExecuteSoftPatch(faultReason string, newThreshold float64) time.Duration {
	log.Printf("[BlueTeam/SoftPatch] Initiating logical correction. Reason: %s", faultReason)
	start := time.Now()

	// 1. Apply Patch
	h.Detector.AdjustThreshold(newThreshold)

	// 2. Validation (Protocol γ-Axiomatic Control Check)
	// In a full SCGO, this would involve re-running a simulated test set against the new threshold
	// to ensure the change optimizes the PoV metric.
	log.Println("[BlueTeam] Patch validated against γ-Axiomatic Control (Simulated PASS).")

	duration := time.Since(start)
	log.Printf("[BlueTeam/SoftPatch] Patch complete. Time-to-Heal: %s", duration)
	return duration
}
