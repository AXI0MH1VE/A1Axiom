package audit

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

// EventType represents different types of auditable events.
type EventType string

const (
	EventDecision      EventType = "decision"
	EventValidation    EventType = "validation"
	EventRateLimit     EventType = "rate_limit"
	EventFaultInjection EventType = "fault_injection"
	EventCompliance    EventType = "compliance"
	EventSecurity      EventType = "security"
	EventPerformance   EventType = "performance"
)

// ComplianceStatus represents the compliance status of an event.
type ComplianceStatus string

const (
	StatusCompliant    ComplianceStatus = "compliant"
	StatusNonCompliant ComplianceStatus = "non_compliant"
	StatusWarning      ComplianceStatus = "warning"
	StatusError        ComplianceStatus = "error"
)

// AuditEvent represents a single auditable event.
type AuditEvent struct {
	ID               string                 `json:"id"`
	Timestamp        time.Time              `json:"timestamp"`
	Type             EventType              `json:"type"`
	Status           ComplianceStatus       `json:"status"`
	Message          string                 `json:"message"`
	Details          map[string]interface{} `json:"details"`
	SourceIP         string                 `json:"source_ip,omitempty"`
	UserAgent        string                 `json:"user_agent,omitempty"`
	RequestID        string                 `json:"request_id,omitempty"`
	ProcessingTimeNS int64                  `json:"processing_time_ns,omitempty"`
	Component        string                 `json:"component"`
	Protocol         string                 `json:"protocol,omitempty"`
}

// Auditor manages comprehensive audit logging for compliance verification.
type Auditor struct {
	mu           sync.RWMutex
	events       []AuditEvent
	outputFile   *os.File
	encoder      *json.Encoder
	maxEvents    int
	eventCounter int64
}

// Config holds auditor configuration.
type Config struct {
	OutputFile   string `json:"output_file"`
	MaxEvents    int    `json:"max_events"`
	EnableConsole bool  `json:"enable_console"`
}

// NewAuditor creates a new auditor instance.
func NewAuditor(config Config) (*Auditor, error) {
	maxEvents := config.MaxEvents
	if maxEvents <= 0 {
		maxEvents = 100000 // Default to 100k events
	}

	file, err := os.OpenFile(config.OutputFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open audit log file: %w", err)
	}

	auditor := &Auditor{
		events:       make([]AuditEvent, 0, maxEvents),
		outputFile:   file,
		encoder:      json.NewEncoder(file),
		maxEvents:    maxEvents,
		eventCounter: 0,
	}

	// Log auditor startup
	auditor.LogEvent(AuditEvent{
		Type:      EventSecurity,
		Status:    StatusCompliant,
		Message:   "Audit system initialized",
		Component: "auditor",
		Details: map[string]interface{}{
			"max_events":   maxEvents,
			"output_file":  config.OutputFile,
			"console_log":  config.EnableConsole,
		},
	})

	log.Printf("Auditor: Initialized with max %d events, output file: %s", maxEvents, config.OutputFile)
	return auditor, nil
}

// LogEvent logs an audit event.
func (a *Auditor) LogEvent(event AuditEvent) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Generate unique event ID
	a.eventCounter++
	event.ID = fmt.Sprintf("evt_%d_%d", time.Now().UnixNano(), a.eventCounter)
	event.Timestamp = time.Now()

	// Add to in-memory store
	a.events = append(a.events, event)

	// Maintain max events limit
	if len(a.events) > a.maxEvents {
		// Remove oldest 10% of events
		removeCount := a.maxEvents / 10
		if removeCount < 100 {
			removeCount = 100
		}
		a.events = a.events[removeCount:]
	}

	// Write to file
	if err := a.encoder.Encode(event); err != nil {
		log.Printf("Auditor: Failed to write event to file: %v", err)
	}

	// Console logging for important events
	if event.Status == StatusError || event.Status == StatusNonCompliant {
		log.Printf("AUDIT [%s] %s: %s - %s",
			event.Status, event.Type, event.Component, event.Message)
	}
}

// LogDecision logs an anomaly detection decision.
func (a *Auditor) LogDecision(decisionID string, isAnomaly bool, zScore float64, latencyNS int64, sourceIP string) {
	status := StatusCompliant
	message := fmt.Sprintf("Decision processed: anomaly=%t, z_score=%.3f", isAnomaly, zScore)

	if latencyNS > 50*1000000 { // 50ms in nanoseconds
		status = StatusWarning
		message += " (high latency)"
	}

	a.LogEvent(AuditEvent{
		Type:             EventDecision,
		Status:           status,
		Message:          message,
		SourceIP:         sourceIP,
		ProcessingTimeNS: latencyNS,
		Component:        "anomaly_detector",
		Protocol:         "γ-Axiomatic Control",
		Details: map[string]interface{}{
			"decision_id": decisionID,
			"is_anomaly":  isAnomaly,
			"z_score":     zScore,
			"latency_ms":  float64(latencyNS) / 1000000,
		},
	})
}

// LogValidation logs a validation event.
func (a *Auditor) LogValidation(success bool, field string, value interface{}, sourceIP string, err error) {
	status := StatusCompliant
	message := fmt.Sprintf("Validation %s for field %s", successStr(success), field)

	if !success {
		status = StatusError
		message = fmt.Sprintf("Validation failed for field %s: %v", field, err)
	}

	a.LogEvent(AuditEvent{
		Type:      EventValidation,
		Status:    status,
		Message:   message,
		SourceIP:  sourceIP,
		Component: "validator",
		Protocol:  "α-IngressGuard",
		Details: map[string]interface{}{
			"field":       field,
			"value":       value,
			"success":     success,
			"error":       errStr(err),
		},
	})
}

// LogRateLimit logs a rate limiting event.
func (a *Auditor) LogRateLimit(allowed bool, sourceIP string, requestID string) {
	status := StatusCompliant
	message := "Request allowed"

	if !allowed {
		status = StatusWarning
		message = "Rate limit exceeded"
	}

	a.LogEvent(AuditEvent{
		Type:      EventRateLimit,
		Status:    status,
		Message:   message,
		SourceIP:  sourceIP,
		RequestID: requestID,
		Component: "rate_limiter",
		Protocol:  "α-IngressGuard",
		Details: map[string]interface{}{
			"allowed": allowed,
		},
	})
}

// LogFaultInjection logs a fault injection event.
func (a *Auditor) LogFaultInjection(faultType string, injected bool, duration time.Duration) {
	status := StatusWarning // Fault injection is expected but notable
	message := fmt.Sprintf("Fault injection: %s", faultType)

	if injected {
		message = fmt.Sprintf("Fault injected: %s for %v", faultType, duration)
	}

	a.LogEvent(AuditEvent{
		Type:      EventFaultInjection,
		Status:    status,
		Message:   message,
		Component: "red_team",
		Protocol:  "β-RedTeam",
		Details: map[string]interface{}{
			"fault_type": faultType,
			"injected":   injected,
			"duration":   duration.String(),
		},
	})
}

// LogCompliance logs a compliance check event.
func (a *Auditor) LogCompliance(protocol string, axiom string, compliant bool, metrics map[string]interface{}) {
	status := StatusCompliant
	message := fmt.Sprintf("%s %s compliance check", protocol, axiom)

	if !compliant {
		status = StatusNonCompliant
		message = fmt.Sprintf("%s %s NON-COMPLIANT", protocol, axiom)
	}

	a.LogEvent(AuditEvent{
		Type:      EventCompliance,
		Status:    status,
		Message:   message,
		Component: "hypervisor",
		Protocol:  protocol,
		Details: map[string]interface{}{
			"axiom":      axiom,
			"compliant":  compliant,
			"metrics":    metrics,
		},
	})
}

// LogPerformance logs a performance metric event.
func (a *Auditor) LogPerformance(component string, metric string, value float64, threshold float64) {
	status := StatusCompliant
	message := fmt.Sprintf("%s %s: %.2f", component, metric, value)

	if value > threshold {
		status = StatusWarning
		message += fmt.Sprintf(" (threshold: %.2f)", threshold)
	}

	a.LogEvent(AuditEvent{
		Type:      EventPerformance,
		Status:    status,
		Message:   message,
		Component: component,
		Details: map[string]interface{}{
			"metric":    metric,
			"value":     value,
			"threshold": threshold,
		},
	})
}

// GetEvents returns recent audit events.
func (a *Auditor) GetEvents(limit int) []AuditEvent {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if limit <= 0 || limit > len(a.events) {
		limit = len(a.events)
	}

	// Return most recent events
	start := len(a.events) - limit
	if start < 0 {
		start = 0
	}

	result := make([]AuditEvent, limit)
	copy(result, a.events[start:])
	return result
}

// GetComplianceReport generates a compliance report.
func (a *Auditor) GetComplianceReport(since time.Time) map[string]interface{} {
	a.mu.RLock()
	defer a.mu.RUnlock()

	report := map[string]interface{}{
		"total_events":       len(a.events),
		"compliant_events":   0,
		"non_compliant_events": 0,
		"warning_events":     0,
		"error_events":       0,
		"protocols":         make(map[string]int),
		"components":        make(map[string]int),
	}

	for _, event := range a.events {
		if event.Timestamp.Before(since) {
			continue
		}

		switch event.Status {
		case StatusCompliant:
			report["compliant_events"] = report["compliant_events"].(int) + 1
		case StatusNonCompliant:
			report["non_compliant_events"] = report["non_compliant_events"].(int) + 1
		case StatusWarning:
			report["warning_events"] = report["warning_events"].(int) + 1
		case StatusError:
			report["error_events"] = report["error_events"].(int) + 1
		}

		// Count by protocol
		if event.Protocol != "" {
			protocols := report["protocols"].(map[string]int)
			protocols[event.Protocol]++
			report["protocols"] = protocols
		}

		// Count by component
		components := report["components"].(map[string]int)
		components[event.Component]++
		report["components"] = components
	}

	return report
}

// Close closes the auditor and flushes any remaining events.
func (a *Auditor) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Log final event
	a.LogEvent(AuditEvent{
		Type:      EventSecurity,
		Status:    StatusCompliant,
		Message:   "Audit system shutdown",
		Component: "auditor",
		Details: map[string]interface{}{
			"total_events_logged": a.eventCounter,
		},
	})

	return a.outputFile.Close()
}

// Helper functions
func successStr(success bool) string {
	if success {
		return "successful"
	}
	return "failed"
}

func errStr(err error) string {
	if err != nil {
		return err.Error()
	}
	return ""
}

// DefaultConfig returns default auditor configuration.
func DefaultConfig() Config {
	return Config{
		OutputFile:    "audit.log",
		MaxEvents:     100000,
		EnableConsole: true,
	}
}