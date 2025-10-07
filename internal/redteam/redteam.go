package redteam

import (
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"
)

// FaultType represents different types of faults that can be injected.
type FaultType string

const (
	FaultLatency        FaultType = "latency"
	FaultMemoryPressure FaultType = "memory_pressure"
	FaultCPUStress      FaultType = "cpu_stress"
	FaultNetworkDelay   FaultType = "network_delay"
	FaultValidationFail FaultType = "validation_fail"
	FaultProcessingFail FaultType = "processing_fail"
)

// FaultConfig represents configuration for a specific fault.
type FaultConfig struct {
	Type         FaultType     `json:"type"`
	Probability  float64       `json:"probability"`  // 0.0 to 1.0
	Duration     time.Duration `json:"duration"`
	Parameters   map[string]interface{} `json:"parameters"`
	Enabled      bool          `json:"enabled"`
}

// RedTeam manages automated fault injection for Protocol β-RedTeam.
type RedTeam struct {
	mu           sync.RWMutex
	faultConfigs map[FaultType]*FaultConfig
	activeFaults map[FaultType]time.Time
	rand         *rand.Rand
}

// NewRedTeam creates a new RedTeam instance.
func NewRedTeam() *RedTeam {
	return &RedTeam{
		faultConfigs: make(map[FaultType]*FaultConfig),
		activeFaults: make(map[FaultType]time.Time),
		rand:         rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// ConfigureFault configures a fault injection pattern.
func (rt *RedTeam) ConfigureFault(config FaultConfig) {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	config.Enabled = true
	rt.faultConfigs[config.Type] = &config

	log.Printf("RedTeam: Configured fault %s with probability %.2f%% for duration %v",
		config.Type, config.Probability*100, config.Duration)
}

// EnableFault enables a specific fault type.
func (rt *RedTeam) EnableFault(faultType FaultType) {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	if config, exists := rt.faultConfigs[faultType]; exists {
		config.Enabled = true
		log.Printf("RedTeam: Enabled fault %s", faultType)
	}
}

// DisableFault disables a specific fault type.
func (rt *RedTeam) DisableFault(faultType FaultType) {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	if config, exists := rt.faultConfigs[faultType]; exists {
		config.Enabled = false
		delete(rt.activeFaults, faultType)
		log.Printf("RedTeam: Disabled fault %s", faultType)
	}
}

// ShouldInjectFault determines if a fault should be injected based on probability.
func (rt *RedTeam) ShouldInjectFault(faultType FaultType) bool {
	rt.mu.RLock()
	config, exists := rt.faultConfigs[faultType]
	rt.mu.RUnlock()

	if !exists || !config.Enabled {
		return false
	}

	// Check if fault is already active and within duration
	rt.mu.RLock()
	if activeTime, isActive := rt.activeFaults[faultType]; isActive {
		if time.Since(activeTime) < config.Duration {
			rt.mu.RUnlock()
			return true
		}
	}
	rt.mu.RUnlock()

	// Roll dice for fault injection
	if rt.rand.Float64() < config.Probability {
		rt.mu.Lock()
		rt.activeFaults[faultType] = time.Now()
		rt.mu.Unlock()

		log.Printf("RedTeam: Injecting fault %s for duration %v", faultType, config.Duration)
		return true
	}

	return false
}

// InjectLatency simulates processing latency (Axiom A-2 stress test).
func (rt *RedTeam) InjectLatency(baseLatency time.Duration) time.Duration {
	if rt.ShouldInjectFault(FaultLatency) {
		rt.mu.RLock()
		config := rt.faultConfigs[FaultLatency]
		rt.mu.RUnlock()

		if multiplier, ok := config.Parameters["multiplier"].(float64); ok {
			return time.Duration(float64(baseLatency) * multiplier)
		}

		// Default: 10x latency
		return baseLatency * 10
	}
	return baseLatency
}

// InjectValidationFault simulates validation failures.
func (rt *RedTeam) InjectValidationFault() error {
	if rt.ShouldInjectFault(FaultValidationFail) {
		return fmt.Errorf("redteam: simulated validation failure")
	}
	return nil
}

// InjectProcessingFault simulates processing failures.
func (rt *RedTeam) InjectProcessingFault() error {
	if rt.ShouldInjectFault(FaultProcessingFail) {
		return fmt.Errorf("redteam: simulated processing failure")
	}
	return nil
}

// GetActiveFaults returns currently active faults.
func (rt *RedTeam) GetActiveFaults() map[FaultType]time.Time {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	active := make(map[FaultType]time.Time)
	for faultType, startTime := range rt.activeFaults {
		rt.mu.RUnlock()
		config := rt.faultConfigs[faultType]
		rt.mu.RLock()

		if config != nil && config.Enabled && time.Since(startTime) < config.Duration {
			active[faultType] = startTime
		}
	}
	return active
}

// GetFaultStats returns statistics about fault injection.
func (rt *RedTeam) GetFaultStats() map[string]interface{} {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	stats := map[string]interface{}{
		"configured_faults": len(rt.faultConfigs),
		"active_faults":    len(rt.GetActiveFaults()),
		"fault_configs":    rt.getFaultConfigsSummary(),
	}

	return stats
}

// getFaultConfigsSummary returns a summary of fault configurations.
func (rt *RedTeam) getFaultConfigsSummary() map[string]interface{} {
	summary := make(map[string]interface{})

	for faultType, config := range rt.faultConfigs {
		summary[string(faultType)] = map[string]interface{}{
			"enabled":     config.Enabled,
			"probability": config.Probability,
			"duration":    config.Duration.String(),
		}
	}

	return summary
}

// SetupDefaultFaults configures a set of default faults for comprehensive testing.
func (rt *RedTeam) SetupDefaultFaults() {
	defaultFaults := []FaultConfig{
		{
			Type:        FaultLatency,
			Probability: 0.05, // 5% chance
			Duration:    time.Minute * 2,
			Parameters:  map[string]interface{}{"multiplier": 5.0},
		},
		{
			Type:        FaultValidationFail,
			Probability: 0.02, // 2% chance
			Duration:    time.Minute,
			Parameters:  map[string]interface{}{},
		},
		{
			Type:        FaultProcessingFail,
			Probability: 0.01, // 1% chance
			Duration:    time.Second * 30,
			Parameters:  map[string]interface{}{},
		},
	}

	for _, fault := range defaultFaults {
		rt.ConfigureFault(fault)
	}

	log.Printf("RedTeam: Configured %d default faults", len(defaultFaults))
}

// CleanupExpiredFaults removes expired faults from active tracking.
func (rt *RedTeam) CleanupExpiredFaults() {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	now := time.Now()
	for faultType, startTime := range rt.activeFaults {
		if config, exists := rt.faultConfigs[faultType]; exists {
			if now.Sub(startTime) >= config.Duration {
				delete(rt.activeFaults, faultType)
				log.Printf("RedTeam: Fault %s expired", faultType)
			}
		}
	}
}

// StartFaultCleanupRoutine starts a goroutine that periodically cleans up expired faults.
func (rt *RedTeam) StartFaultCleanupRoutine() {
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			rt.CleanupExpiredFaults()
		}
	}()

	log.Println("RedTeam: Started fault cleanup routine")
}