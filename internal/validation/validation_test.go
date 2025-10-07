package validation

import (
	"testing"
	"time"
)

func TestDataPointValidator_ValidateDataPoint(t *testing.T) {
	now := time.Now()
	config := Config{
		MaxValue:      1000.0,
		MinValue:      -1000.0,
		MaxTimestamp:  now.Unix() + 3600,
		MinTimestamp:  now.Unix() - 86400,
		AllowedSource: "*",
	}

	validator := NewDataPointValidator(config)

	testCases := []struct {
		name        string
		timestamp   int64
		value       float64
		sourceIP    string
		expectError bool
	}{
		{
			name:        "Valid data point",
			timestamp:   now.Unix(),
			value:       42.5,
			sourceIP:    "192.168.1.1",
			expectError: false,
		},
		{
			name:        "Invalid timestamp - too old",
			timestamp:   now.Unix() - 86401, // Older than minimum
			value:       42.5,
			sourceIP:    "192.168.1.1",
			expectError: true,
		},
		{
			name:        "Invalid timestamp - too future",
			timestamp:   now.Unix() + 3601, // Newer than maximum
			value:       42.5,
			sourceIP:    "192.168.1.1",
			expectError: true,
		},
		{
			name:        "Invalid value - too high",
			timestamp:   now.Unix(),
			value:       1001.0, // Higher than maximum
			sourceIP:    "192.168.1.1",
			expectError: true,
		},
		{
			name:        "Invalid value - too low",
			timestamp:   now.Unix(),
			value:       -1001.0, // Lower than minimum
			sourceIP:    "192.168.1.1",
			expectError: true,
		},
		{
			name:        "Invalid timestamp - zero",
			timestamp:   0,
			value:       42.5,
			sourceIP:    "192.168.1.1",
			expectError: true,
		},
		{
			name:        "Invalid timestamp - negative",
			timestamp:   -100,
			value:       42.5,
			sourceIP:    "192.168.1.1",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validator.ValidateDataPoint(tc.timestamp, tc.value, tc.sourceIP)

			if tc.expectError && err == nil {
				t.Error("Expected validation error, but got none")
			}

			if !tc.expectError && err != nil {
				t.Errorf("Expected no validation error, but got: %v", err)
			}
		})
	}
}

func TestDataPointValidator_SourceIPValidation(t *testing.T) {
	now := time.Now().Unix()

	testCases := []struct {
		name          string
		allowedSource string
		sourceIP      string
		expectError   bool
	}{
		{
			name:          "Allow all sources",
			allowedSource: "*",
			sourceIP:      "192.168.1.1",
			expectError:   false,
		},
		{
			name:          "Exact IP match",
			allowedSource: "192.168.1.100",
			sourceIP:      "192.168.1.100",
			expectError:   false,
		},
		{
			name:          "Exact IP mismatch",
			allowedSource: "192.168.1.100",
			sourceIP:      "192.168.1.101",
			expectError:   true,
		},
		{
			name:          "CIDR match",
			allowedSource: "192.168.1.0/24",
			sourceIP:      "192.168.1.50",
			expectError:   false,
		},
		{
			name:          "CIDR mismatch",
			allowedSource: "192.168.1.0/24",
			sourceIP:      "192.168.2.50",
			expectError:   true,
		},
		{
			name:          "Wildcard match",
			allowedSource: "192.168.*",
			sourceIP:      "192.168.1.1",
			expectError:   false,
		},
		{
			name:          "Wildcard mismatch",
			allowedSource: "192.168.*",
			sourceIP:      "10.0.1.1",
			expectError:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := Config{
				MaxValue:      1000.0,
				MinValue:      -1000.0,
				MaxTimestamp:  now + 3600,
				MinTimestamp:  now - 86400,
				AllowedSource: tc.allowedSource,
			}

			validator := NewDataPointValidator(config)
			err := validator.ValidateDataPoint(now, 42.5, tc.sourceIP)

			if tc.expectError && err == nil {
				t.Error("Expected validation error, but got none")
			}

			if !tc.expectError && err != nil {
				t.Errorf("Expected no validation error, but got: %v", err)
			}
		})
	}
}

func TestDataPointValidator_FutureTimestampTolerance(t *testing.T) {
	now := time.Now().Unix()
	config := Config{
		MaxValue:      1000.0,
		MinValue:      -1000.0,
		MaxTimestamp:  now + 300, // 5 minutes tolerance
		MinTimestamp:  now - 86400,
		AllowedSource: "*",
	}

	validator := NewDataPointValidator(config)

	// Test timestamp within tolerance
	err := validator.ValidateDataPoint(now+100, 42.5, "192.168.1.1")
	if err != nil {
		t.Errorf("Expected no error for timestamp within tolerance, got: %v", err)
	}

	// Test timestamp beyond tolerance
	err = validator.ValidateDataPoint(now+400, 42.5, "192.168.1.1")
	if err == nil {
		t.Error("Expected error for timestamp beyond tolerance, but got none")
	}
}

func TestDataPointValidator_EdgeValues(t *testing.T) {
	now := time.Now().Unix()
	config := Config{
		MaxValue:      100.0,
		MinValue:      -100.0,
		MaxTimestamp:  now + 3600,
		MinTimestamp:  now - 86400,
		AllowedSource: "*",
	}

	validator := NewDataPointValidator(config)

	testCases := []struct {
		name      string
		timestamp int64
		value     float64
	}{
		{"Maximum value", now, 100.0},
		{"Minimum value", now, -100.0},
		{"Zero value", now, 0.0},
		{"Maximum timestamp", now + 3600, 50.0},
		{"Minimum timestamp", now - 86400, 50.0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validator.ValidateDataPoint(tc.timestamp, tc.value, "192.168.1.1")
			if err != nil {
				t.Errorf("Expected no error for edge value, got: %v", err)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	now := time.Now().Unix()

	if config.MaxValue <= 0 {
		t.Error("Expected positive max value in default config")
	}

	if config.MinValue >= 0 {
		t.Error("Expected negative min value in default config")
	}

	if config.MaxTimestamp <= now {
		t.Error("Expected max timestamp to be in the future")
	}

	if config.MinTimestamp >= now {
		t.Error("Expected min timestamp to be in the past")
	}

	if config.AllowedSource != "*" {
		t.Error("Expected wildcard allowed source in default config")
	}
}

func TestValidationErrors_Error(t *testing.T) {
	errors := []ValidationError{
		{Field: "timestamp", Value: "0", Message: "must be positive"},
		{Field: "value", Value: "999999", Message: "exceeds maximum"},
	}

	validationErrors := NewValidationErrors(errors)
	errorMsg := validationErrors.Error()

	if !contains(errorMsg, "timestamp") {
		t.Error("Expected error message to contain 'timestamp'")
	}

	if !contains(errorMsg, "value") {
		t.Error("Expected error message to contain 'value'")
	}

	if !contains(errorMsg, "must be positive") {
		t.Error("Expected error message to contain 'must be positive'")
	}

	if !contains(errorMsg, "exceeds maximum") {
		t.Error("Expected error message to contain 'exceeds maximum'")
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || containsRecursive(s, substr))
}

func containsRecursive(s, substr string) bool {
	if len(s) < len(substr) {
		return false
	}
	if s[:len(substr)] == substr {
		return true
	}
	return containsRecursive(s[1:], substr)
}