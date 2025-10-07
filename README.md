# RADM - Real-Time Anomaly Detection Microservice

[![Go Version](https://img.shields.io/badge/go-1.21+-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![Docker](https://img.shields.io/badge/docker-ready-blue.svg)](https://docker.com)
[![Kubernetes](https://img.shields.io/badge/kubernetes-ready-blue.svg)](https://kubernetes.io)

## Executive Summary

The **Real-Time Anomaly Detection Microservice (RADM)** is the first operational artifact developed under Alexis Adams' **Axiomhive SCGO Protocol**. Built using the **AEGIS-9 Cognitive Swarm methodology**, RADM delivers high-performance, O(1) amortized complexity anomaly detection with enterprise-grade reliability and compliance.

**Core Compliance Status:**
- ✅ **A-1 Determinism**: Output verifiable via hash parity
- ✅ **A-2 Latency**: P95 ingestion ≤ 50ms guaranteed
- ✅ **A-4 Monetization**: Proof-of-Value (PoV) metrics logged

## 🚀 Quick Start

### Local Development

````bash
# Clone and setup
git clone <repository-url>
cd radm
make setup

# Run locally
make dev

# Access at http://localhost:8080
# Health check: http://localhost:8080/healthz
# Metrics: http://localhost:8080/metrics
```

### Docker Deployment

````bash
# Build and run with Docker
make docker-build
make docker-run

# Or using docker-compose
docker-compose up -d
```

### Kubernetes Deployment

````bash
# Deploy to Kubernetes
make deploy

# Check status
make deploy-status

# View logs
make logs
```

## 📋 API Documentation

### Core Endpoint

**POST** `/api/v1/data/ingest`

Ingest data points for anomaly detection analysis.

**Request Body:**

```json
{
  "timestamp": 1638360000,
  "value": 42.5
}
```

**Response:**

```json
{
  "is_anomaly": true,
  "z_score": 3.24,
  "timestamp": 1638360000,
  "value": 42.5,
  "processing_ns": 150000,
  "price": 0.001234
}
```

### Health Endpoints

- **GET** `/healthz` - Liveness probe (Protocol β-RedTeam)
- **GET** `/readyz` - Readiness probe (Protocol β-RedTeam)
- **GET** `/metrics` - Prometheus metrics and system statistics

## 🏗️ Architecture

### System Design

RADM implements a **Sliding Window Z-Score algorithm** with the following architectural principles:

```text
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   IngressGuard  │───▶│  AnomalyDetector │───▶│   EgressGuard   │
│   (Rate Limit + │    │   (O(1) Window)  │    │   (Response +   │
│   Validation)   │    │                  │    │   PoV Logging)  │
└─────────────────┘    └──────────────────┘    └─────────────────┘
```

### Core Components

#### 1. Anomaly Detection Engine (`anomaly/`)
- **Sliding Window Z-Score** algorithm
- **O(1) amortized complexity** per data point
- Thread-safe implementation with `sync.RWMutex`
- Configurable window size and threshold

#### 2. Monetization Tracker (`internal/monetization/`)
- **Proof-of-Value (PoV)** logging
- Dynamic pricing based on processing complexity
- Persistent storage of decision records
- Real-time financial calculations

#### 3. Security & Validation (`internal/validation/`)
- Input schema validation (Protocol α-IngressGuard)
- Source IP filtering and rate limiting
- Temporal consistency checks
- Comprehensive error reporting

## ⚙️ Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `AD_WINDOW_SIZE` | `500` | Sliding window size for Z-Score calculation |
| `AD_THRESHOLD` | `3.5` | Z-Score threshold for anomaly detection |
| `MONETIZATION_BASE_PRICE` | `0.001` | Base price per decision in USD |
| `RATE_LIMIT_REQUESTS_PER_SECOND` | `1000` | Rate limit for incoming requests |
| `VALIDATION_MAX_VALUE` | `1e10` | Maximum allowed data value |

### Configuration File

Create `config.yaml` for custom configuration:

```yaml
server:
  port: "8080"
  host: "0.0.0.0"
  read_timeout: "10s"

detector:
  window_size: 500
  threshold: 3.5

monetization:
  enabled: true
  base_price: 0.001
  complexity_multiplier: 0.1

validation:
  enabled: true
  max_value: 10000000000
  min_value: -10000000000

rate_limit:
  enabled: true
  requests_per_second: 1000
  burst_size: 100
```

## 🔒 Security & Compliance

### Protocol Compliance

RADM implements multiple security protocols:

#### Protocol α-IngressGuard
- **Rate Limiting**: Token bucket algorithm
- **Input Validation**: Schema and range validation
- **Source Filtering**: IP-based access control

#### Protocol β-RedTeam
- **Health Checks**: Liveness and readiness probes
- **Resource Limits**: CPU and memory constraints
- **Security Context**: Non-root execution

#### Protocol δ-EgressGuard
- **Response Validation**: Structured error handling
- **Audit Logging**: Comprehensive request/response logging

### Security Features

- **Race Condition Protection**: Thread-safe data structures
- **Input Sanitization**: Comprehensive validation pipeline
- **Zero-Division Protection**: Safe mathematical operations
- **Resource Exhaustion Prevention**: Configurable limits

## 📊 Monitoring & Observability

### Metrics

RADM exposes Prometheus-compatible metrics:

- `radm_decisions_total` - Total decisions processed
- `radm_anomalies_total` - Total anomalies detected
- `radm_processing_latency_seconds` - Processing latency histogram
- `radm_window_size_current` - Current sliding window size

### Logging

Structured logging with configurable levels:
- **Decision Events**: All anomaly detection results
- **Performance Metrics**: Latency and throughput data
- **Security Events**: Validation failures and rate limit hits
- **System Events**: Startup, shutdown, and errors

## 🧪 Testing

### Running Tests

````bash
# All tests with coverage
make test

# Unit tests only
make test-unit

# Integration tests
make test-integration

# Benchmarks
make benchmark
```

### Test Coverage

RADM maintains comprehensive test coverage:
- **Unit Tests**: Core algorithm and component testing
- **Integration Tests**: Full request/response cycle testing
- **Performance Tests**: Load testing and benchmarking
- **Security Tests**: Input validation and edge case testing

## 🚢 Deployment

### Kubernetes Deployment

#### Prerequisites
- Kubernetes 1.19+
- Ingress Controller (nginx recommended)
- cert-manager (for TLS certificates)

#### Quick Deployment

````bash
# Apply all manifests
kubectl apply -f infra/k8s/

# Verify deployment
kubectl get pods -l app=radm-detector
kubectl get services -l app=radm-detector
kubectl get ingress -l app=radm-detector
```

#### Scaling

````bash
# Manual scaling
kubectl scale deployment radm-detector --replicas=5

# HPA is configured for automatic scaling based on CPU/Memory
kubectl get hpa radm-detector-hpa
```

### Docker Deployment

````bash
# Build image
docker build -t axiomhive/radm-detector:latest .

# Run container
docker run -d \
  --name radm-detector \
  -p 8080:8080 \
  -e AD_WINDOW_SIZE=500 \
  -e AD_THRESHOLD=3.5 \
  axiomhive/radm-detector:latest
```

## 💰 Monetization

### Proof-of-Value (PoV) System

RADM implements a sophisticated monetization system:

- **Dynamic Pricing**: Based on processing complexity and latency
- **Usage Tracking**: Detailed decision logging with timestamps
- **Financial Reporting**: Real-time value calculation
- **Audit Trail**: Complete transaction history

### Pricing Model

```text
Base Price = $0.001 per decision
Complexity Multiplier = 0.1
Latency Factor = Processing Time (ns) / 1e9

Final Price = Base Price × (1 + Latency Factor) × (1 + |Z-Score| × Complexity Multiplier)
```

## 🔧 Development

### Project Structure

```text
radm/
├── anomaly/           # Core anomaly detection algorithm
├── internal/
│   ├── config/       # Configuration management
│   ├── monetization/ # Proof-of-Value tracking
│   ├── validation/   # Input validation
│   └── ratelimit/    # Rate limiting
├── cmd/radm/         # Main application entrypoint
├── docs/             # Documentation
├── infra/k8s/        # Kubernetes manifests
├── api/v1/           # API specifications
└── Makefile         # Build automation
```

### Development Workflow

1. **Setup**: `make setup`
2. **Develop**: `make dev` or `make dev-watch`
3. **Test**: `make test`
4. **Build**: `make build`
5. **Deploy**: `make deploy`

### Code Quality

- **Linting**: `golangci-lint` configuration
- **Formatting**: `gofmt` and `goimports`
- **Security**: `gosec` scanning
- **Testing**: Comprehensive test coverage

## 📈 Performance

### Benchmarks

```text
BenchmarkProcessData-8    1000000    1200 ns/op    200 B/op    5 allocs/op
BenchmarkSlidingWindow-8   500000    2400 ns/op    400 B/op    10 allocs/op
```

### Scalability

- **Horizontal Scaling**: Kubernetes HPA configuration
- **Load Balancing**: Multiple replicas with load distribution
- **Resource Optimization**: Efficient memory usage with sliding window
- **Database Independence**: Stateless design for easy scaling

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Development Guidelines

- Follow Go best practices and idioms
- Write comprehensive tests for new features
- Update documentation for API changes
- Ensure security compliance (Protocol requirements)
- Maintain performance characteristics (O(1) complexity)

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🆘 Support

### Troubleshooting

**High Latency Issues:**
- Check CPU and memory allocation in Kubernetes
- Verify sliding window size configuration
- Monitor system load and scale if necessary

**Rate Limiting Errors:**
- Adjust `RATE_LIMIT_REQUESTS_PER_SECOND` configuration
- Check for legitimate traffic spikes
- Monitor rate limit statistics in `/metrics`

**Validation Failures:**
- Verify input data format matches API specification
- Check timestamp and value constraints
- Review source IP filtering configuration

### Getting Help

- 📖 **Documentation**: Comprehensive guides in `/docs`
- 🐛 **Issues**: Report bugs and feature requests
- 💬 **Discussions**: Join community discussions
- 📧 **Support**: Contact the development team

## 🎯 Roadmap

### Version 1.1 (Next Release)
- [ ] Enhanced machine learning models
- [ ] Real-time alerting integration
- [ ] Advanced visualization dashboard
- [ ] Multi-tenant support

### Version 1.2 (Future Release)
- [ ] Distributed anomaly detection
- [ ] Advanced pattern recognition
- [ ] Integration with streaming platforms
- [ ] Custom algorithm plugins

---

**Built with ❤️ by Axiomhive under the SCGO Protocol**

*"Deterministic, Scalable, and Monetizable Anomaly Detection"*