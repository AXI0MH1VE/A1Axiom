package validation

import (
	"errors"
	"fmt"
	"log"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"anomaly"
)

// ValidationError represents a validation failure with context.
type ValidationError struct {
	Field   string `json:"field"`
	Value   string `json:"value"`
	Message string `json:"message"`
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("validation failed for field '%s' with value '%s': %s", e.Field, e.Value, e.Message)
}

// DataPointRequest represents the incoming data point for validation.
type DataPointRequest struct {
	Timestamp int64   `json:"timestamp" validate:"required,gt=0"`
	Value     float64 `json:"value" validate:"required"`
	SourceIP  string  `json:"source_ip" validate:"required,ip"`
}

// DataPointValidator handles validation for DataPoint structures.
type DataPointValidator struct {
	maxValue      float64
	minValue      float64
	maxTimestamp  int64
	minTimestamp  int64
	allowedSource string
	validator     *validator.Validate
}

// Config holds validation configuration.
type Config struct {
	MaxValue      float64 `json:"max_value"`
	MinValue      float64 `json:"min_value"`
	MaxTimestamp  int64   `json:"max_timestamp"`
	MinTimestamp  int64   `json:"min_timestamp"`
	AllowedSource string  `json:"allowed_source"`
}

// NewDataPointValidator creates a new validator with the given configuration.
func NewDataPointValidator(config Config) *DataPointValidator {
	validate := validator.New()
	return &DataPointValidator{
		maxValue:      config.MaxValue,
		minValue:      config.MinValue,
		maxTimestamp:  config.MaxTimestamp,
		minTimestamp:  config.MinTimestamp,
		allowedSource: config.AllowedSource,
		validator:     validate,
	}
}

// ValidateDataPoint performs comprehensive validation on a data point.
func (v *DataPointValidator) ValidateDataPoint(timestamp int64, value float64, sourceIP string) error {
	var errors []ValidationError

	// Use validator/v10 for basic schema compliance
	req := DataPointRequest{
		Timestamp: timestamp,
		Value:     value,
		SourceIP:  sourceIP,
	}
	if err := v.validator.Struct(req); err != nil {
		validationErrors := err.(validator.ValidationErrors)
		for _, fieldErr := range validationErrors {
			errors = append(errors, ValidationError{
				Field:   strings.ToLower(fieldErr.Field()),
				Value:   fmt.Sprintf("%v", fieldErr.Value()),
				Message: fieldErr.Error(),
			})
		}
	}

	// Timestamp validation
	if timestamp <= 0 {
		errors = append(errors, ValidationError{
			Field:   "timestamp",
			Value:   strconv.FormatInt(timestamp, 10),
			Message: "timestamp must be positive",
		})
	}

	if timestamp < v.minTimestamp {
		errors = append(errors, ValidationError{
			Field:   "timestamp",
			Value:   strconv.FormatInt(timestamp, 10),
			Message: fmt.Sprintf("timestamp below minimum threshold %d", v.minTimestamp),
		})
	}

	if timestamp > v.maxTimestamp {
		errors = append(errors, ValidationError{
			Field:   "timestamp",
			Value:   strconv.FormatInt(timestamp, 10),
			Message: fmt.Sprintf("timestamp exceeds maximum threshold %d", v.maxTimestamp),
		})
	}

	// Value validation
	if value > v.maxValue {
		errors = append(errors, ValidationError{
			Field:   "value",
			Value:   strconv.FormatFloat(value, 'f', -1, 64),
			Message: fmt.Sprintf("value exceeds maximum threshold %.2f", v.maxValue),
		})
	}

	if value < v.minValue {
		errors = append(errors, ValidationError{
			Field:   "value",
			Value:   strconv.FormatFloat(value, 'f', -1, 64),
			Message: fmt.Sprintf("value below minimum threshold %.2f", v.minValue),
		})
	}

	// Source IP validation (if configured)
	if v.allowedSource != "" {
		if !v.isValidSourceIP(sourceIP, v.allowedSource) {
			errors = append(errors, ValidationError{
				Field:   "source_ip",
				Value:   sourceIP,
				Message: fmt.Sprintf("source IP not in allowed range: %s", v.allowedSource),
			})
		}
	}

	// Temporal consistency check (basic)
	now := time.Now().Unix()
	if timestamp > now+300 { // Allow 5 minutes future tolerance
		errors = append(errors, ValidationError{
			Field:   "timestamp",
			Value:   strconv.FormatInt(timestamp, 10),
			Message: "timestamp is too far in the future",
		})
	}

	if len(errors) > 0 {
		return NewValidationErrors(errors)
	}

	return nil
}

// isValidSourceIP checks if the source IP matches the allowed pattern.
func (v *DataPointValidator) isValidSourceIP(sourceIP, allowedPattern string) bool {
	if allowedPattern == "*" {
		return true
	}

	// Support CIDR notation and wildcard patterns
	if strings.Contains(allowedPattern, "/") {
		_, cidr, err := net.ParseCIDR(allowedPattern)
		if err != nil {
			return false
		}
		ip := net.ParseIP(sourceIP)
		return cidr.Contains(ip)
	}

	// Support wildcard patterns like "192.168.1.*"
	if strings.Contains(allowedPattern, "*") {
		pattern := strings.ReplaceAll(allowedPattern, "*", ".*")
		matched, _ := regexp.MatchString(pattern, sourceIP)
		return matched
	}

	// Exact match
	return sourceIP == allowedPattern
}

// ValidationErrors represents multiple validation errors.
type ValidationErrors struct {
	Errors []ValidationError `json:"errors"`
}

func (e ValidationErrors) Error() string {
	if len(e.Errors) == 0 {
		return "validation failed"
	}

	var messages []string
	for _, err := range e.Errors {
		messages = append(messages, err.Error())
	}
	return strings.Join(messages, "; ")
}

// NewValidationErrors creates a new ValidationErrors instance.
func NewValidationErrors(errors []ValidationError) *ValidationErrors {
	return &ValidationErrors{
		Errors: errors,
	}
}

// Validate is the global validator instance.
var validate *validator.Validate

func init() {
	validate = validator.New()
}

// ValidateDataPoint validates a data point using validator/v10.
// This function enforces Protocol α-IngressGuard.
func ValidateDataPoint(dp anomaly.DataPoint) error {
	if err := validate.Struct(dp); err != nil {
		log.Printf("[α-IngressGuard] Validation Failure: %v", err)
		return err
	}
	return nil
}

// ValidateDataPointFromAnomaly validates an anomaly.DataPoint using validator/v10.
// This function enforces Protocol α-IngressGuard for the core anomaly.DataPoint struct.
func ValidateDataPointFromAnomaly(dp anomaly.DataPoint, sourceIP string) error {
	req := DataPointRequest{
		Timestamp: dp.Timestamp,
		Value:     dp.Value,
		SourceIP:  sourceIP,
	}
	return ValidateDataPoint(req)
}

// DefaultConfig returns a default validation configuration.
func DefaultConfig() Config {
	now := time.Now()
	return Config{
		MaxValue:     1e10,  // Very high maximum
		MinValue:     -1e10, // Very low minimum
		MaxTimestamp: now.Unix() + 3600, // Allow 1 hour future
		MinTimestamp: now.Unix() - 86400, // Allow 24 hours past
		AllowedSource: "*", // Allow all sources by default
	}
}