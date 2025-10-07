package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all configuration for the RADM service.
type Config struct {
	Server   ServerConfig   `json:"server"`
	Detector DetectorConfig `json:"detector"`
	Monetization MonetizationConfig `json:"monetization"`
	Validation ValidationConfig `json:"validation"`
	RateLimit RateLimitConfig `json:"rate_limit"`
}

// ServerConfig holds server-related configuration.
type ServerConfig struct {
	Port         string `json:"port"`
	Host         string `json:"host"`
	ReadTimeout  time.Duration `json:"read_timeout"`
	WriteTimeout time.Duration `json:"write_timeout"`
	IdleTimeout  time.Duration `json:"idle_timeout"`
}

// DetectorConfig holds anomaly detector configuration.
type DetectorConfig struct {
	WindowSize int     `json:"window_size"`
	Threshold  float64 `json:"threshold"`
}

// MonetizationConfig holds monetization tracking configuration.
type MonetizationConfig struct {
	BasePrice            float64 `json:"base_price"`
	ComplexityMultiplier float64 `json:"complexity_multiplier"`
	OutputFile           string  `json:"output_file"`
	Enabled              bool    `json:"enabled"`
}

// ValidationConfig holds input validation configuration.
type ValidationConfig struct {
	MaxValue      float64 `json:"max_value"`
	MinValue      float64 `json:"min_value"`
	MaxTimestamp  int64   `json:"max_timestamp"`
	MinTimestamp  int64   `json:"min_timestamp"`
	AllowedSource string  `json:"allowed_source"`
	Enabled       bool    `json:"enabled"`
}

// RateLimitConfig holds rate limiting configuration.
type RateLimitConfig struct {
	RequestsPerSecond int64 `json:"requests_per_second"`
	BurstSize         int64 `json:"burst_size"`
	Enabled           bool  `json:"enabled"`
}

// Load loads configuration from environment variables and files.
func Load() (*Config, error) {
	config := DefaultConfig()

	// Server configuration
	if port := os.Getenv("SERVER_PORT"); port != "" {
		config.Server.Port = port
	}
	if host := os.Getenv("SERVER_HOST"); host != "" {
		config.Server.Host = host
	}
	if readTimeout := os.Getenv("SERVER_READ_TIMEOUT"); readTimeout != "" {
		if d, err := time.ParseDuration(readTimeout); err == nil {
			config.Server.ReadTimeout = d
		}
	}
	if writeTimeout := os.Getenv("SERVER_WRITE_TIMEOUT"); writeTimeout != "" {
		if d, err := time.ParseDuration(writeTimeout); err == nil {
			config.Server.WriteTimeout = d
		}
	}
	if idleTimeout := os.Getenv("SERVER_IDLE_TIMEOUT"); idleTimeout != "" {
		if d, err := time.ParseDuration(idleTimeout); err == nil {
			config.Server.IdleTimeout = d
		}
	}

	// Detector configuration
	if windowSize := os.Getenv("AD_WINDOW_SIZE"); windowSize != "" {
		if ws, err := strconv.Atoi(windowSize); err == nil {
			config.Detector.WindowSize = ws
		}
	}
	if threshold := os.Getenv("AD_THRESHOLD"); threshold != "" {
		if t, err := strconv.ParseFloat(threshold, 64); err == nil {
			config.Detector.Threshold = t
		}
	}

	// Monetization configuration
	if basePrice := os.Getenv("MONETIZATION_BASE_PRICE"); basePrice != "" {
		if bp, err := strconv.ParseFloat(basePrice, 64); err == nil {
			config.Monetization.BasePrice = bp
		}
	}
	if complexityMultiplier := os.Getenv("MONETIZATION_COMPLEXITY_MULTIPLIER"); complexityMultiplier != "" {
		if cm, err := strconv.ParseFloat(complexityMultiplier, 64); err == nil {
			config.Monetization.ComplexityMultiplier = cm
		}
	}
	if outputFile := os.Getenv("MONETIZATION_OUTPUT_FILE"); outputFile != "" {
		config.Monetization.OutputFile = outputFile
	}
	if enabled := os.Getenv("MONETIZATION_ENABLED"); enabled != "" {
		config.Monetization.Enabled = enabled == "true"
	}

	// Validation configuration
	if maxValue := os.Getenv("VALIDATION_MAX_VALUE"); maxValue != "" {
		if mv, err := strconv.ParseFloat(maxValue, 64); err == nil {
			config.Validation.MaxValue = mv
		}
	}
	if minValue := os.Getenv("VALIDATION_MIN_VALUE"); minValue != "" {
		if mv, err := strconv.ParseFloat(minValue, 64); err == nil {
			config.Validation.MinValue = mv
		}
	}
	if maxTimestamp := os.Getenv("VALIDATION_MAX_TIMESTAMP"); maxTimestamp != "" {
		if mt, err := strconv.ParseInt(maxTimestamp, 10, 64); err == nil {
			config.Validation.MaxTimestamp = mt
		}
	}
	if minTimestamp := os.Getenv("VALIDATION_MIN_TIMESTAMP"); minTimestamp != "" {
		if mt, err := strconv.ParseInt(minTimestamp, 10, 64); err == nil {
			config.Validation.MinTimestamp = mt
		}
	}
	if allowedSource := os.Getenv("VALIDATION_ALLOWED_SOURCE"); allowedSource != "" {
		config.Validation.AllowedSource = allowedSource
	}
	if enabled := os.Getenv("VALIDATION_ENABLED"); enabled != "" {
		config.Validation.Enabled = enabled == "true"
	}

	// Rate limit configuration
	if requestsPerSecond := os.Getenv("RATE_LIMIT_REQUESTS_PER_SECOND"); requestsPerSecond != "" {
		if rps, err := strconv.ParseInt(requestsPerSecond, 10, 64); err == nil {
			config.RateLimit.RequestsPerSecond = rps
		}
	}
	if burstSize := os.Getenv("RATE_LIMIT_BURST_SIZE"); burstSize != "" {
		if bs, err := strconv.ParseInt(burstSize, 10, 64); err == nil {
			config.RateLimit.BurstSize = bs
		}
	}
	if enabled := os.Getenv("RATE_LIMIT_ENABLED"); enabled != "" {
		config.RateLimit.Enabled = enabled == "true"
	}

	return config, nil
}

// DefaultConfig returns a default configuration.
func DefaultConfig() *Config {
	now := time.Now()
	return &Config{
		Server: ServerConfig{
			Port:         "8080",
			Host:         "0.0.0.0",
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		Detector: DetectorConfig{
			WindowSize: 500,
			Threshold:  3.5,
		},
		Monetization: MonetizationConfig{
			BasePrice:            0.001,
			ComplexityMultiplier: 0.1,
			OutputFile:           "pov_records.jsonl",
			Enabled:              true,
		},
		Validation: ValidationConfig{
			MaxValue:      1e10,
			MinValue:      -1e10,
			MaxTimestamp:  now.Unix() + 3600,
			MinTimestamp:  now.Unix() - 86400,
			AllowedSource: "*",
			Enabled:       true,
		},
		RateLimit: RateLimitConfig{
			RequestsPerSecond: 1000,
			BurstSize:         100,
			Enabled:           true,
		},
	}
}

// Save saves the configuration to a file.
func (c *Config) Save(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(c); err != nil {
		return fmt.Errorf("failed to encode config: %w", err)
	}

	return nil
}

// Validate validates the configuration for consistency.
func (c *Config) Validate() error {
	if c.Server.Port == "" {
		return fmt.Errorf("server port cannot be empty")
	}

	if c.Detector.WindowSize <= 0 {
		return fmt.Errorf("detector window size must be positive")
	}

	if c.Detector.Threshold < 0 {
		return fmt.Errorf("detector threshold cannot be negative")
	}

	if c.Monetization.BasePrice < 0 {
		return fmt.Errorf("monetization base price cannot be negative")
	}

	if c.RateLimit.RequestsPerSecond < 0 {
		return fmt.Errorf("rate limit requests per second cannot be negative")
	}

	if c.RateLimit.BurstSize < 0 {
		return fmt.Errorf("rate limit burst size cannot be negative")
	}

	return nil
}