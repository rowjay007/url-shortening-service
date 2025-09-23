# URL Shortening Service 🔗

A high-performance, enterprise-grade URL shortening service built with Go, Gin, and PocketBase. This service provides a robust RESTful API for shortening URLs, tracking statistics, and managing short codes with enterprise-level reliability and a built-in admin dashboard.

## 🚀 Features

- **RESTful API**: Complete CRUD operations for URL shortening
- **Unique Short Codes**: Cryptographically secure, collision-resistant base62 short codes
- **Statistics Tracking**: Real-time access count tracking with atomic updates
- **Enterprise Architecture**: Clean architecture with proper separation of concerns
- **Database Persistence**: PocketBase with SQLite backend and built-in admin UI
- **Concurrency Safe**: Thread-safe operations with proper error handling
- **Docker Support**: Complete containerization with Docker Compose
- **Health Checks**: Built-in health monitoring endpoints
- **Middleware Stack**: Comprehensive logging, recovery, and CORS support
- **Configuration Management**: Environment-based configuration system
- **Unit Testing**: Comprehensive test coverage for critical components

## 📋 API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/api/v1/shorten` | Create a new short URL |
| `GET` | `/api/v1/shorten/:shortCode` | Retrieve original URL (increments access count) |
| `PUT` | `/api/v1/shorten/:shortCode` | Update existing short URL |
| `DELETE` | `/api/v1/shorten/:shortCode` | Delete short URL |
| `GET` | `/api/v1/shorten/:shortCode/stats` | Get access statistics |
| `GET` | `/health` | Health check endpoint |

## 🛠️ Technology Stack

- **Language**: Go 1.21+
- **Web Framework**: Gin
- **Database**: PocketBase (SQLite-based backend with admin UI)
- **Data Access**: PocketBase SDK
- **Containerization**: Docker & Docker Compose
- **Testing**: Go testing package with mocks
- **Configuration**: Environment variables with godotenv

## 🏗️ Architecture

```
cmd/
└── server/           # Application entrypoint
internal/
├── config/          # Configuration management
├── database/        # PocketBase client initialization
├── handlers/        # HTTP request handlers (controllers)
├── middleware/      # HTTP middleware (logging, recovery, CORS)
├── models/         # Data models and DTOs
├── repository/     # Data access layer
├── services/       # Business logic layer
└── utils/          # Utility functions (short code generation)
scripts/            # Database initialization scripts
```

## 🚦 Quick Start

### Prerequisites

- Go 1.21 or higher
- Docker and Docker Compose
- PocketBase (automatically managed via Docker)

### Option 1: Docker Compose (Recommended)

1. **Clone the repository**
   ```bash
   git clone <repository-url>
   cd url-shortening-service
   ```

2. **Start the services**
   ```bash
   task docker:run
   ```

   This will start:
   - PocketBase backend on port 8090 (with admin UI at http://localhost:8090/_/)
   - URL shortening service on port 8080
   - Redis (for future caching) on port 6379

3. **Set up PocketBase Collection**
   - Visit http://localhost:8090/_/ to access the PocketBase admin UI
   - Create your first superuser account when prompted
   - Create a new "Base" collection named `short_urls` with the following fields:
     - `url` (Text, required)
     - `short_code` (Text, required, unique)
     - `access_count` (Number, default: 0)

4. **Test the service**
   ```bash
   curl -X POST http://localhost:8080/api/v1/shorten \
     -H "Content-Type: application/json" \
     -d '{"url": "https://example.com/very/long/url"}'
   ```

### Option 2: Local Development

1. **Initial setup**
   ```bash
   task setup
   ```

2. **Set up PocketBase**
   ```bash
   task db:up  # This starts PocketBase in Docker
   ```

3. **Run the application**
   ```bash
   task run
   ```

## 📝 API Usage Examples

### Create Short URL
```bash
curl -X POST http://localhost:8080/api/v1/shorten \
  -H "Content-Type: application/json" \
  -d '{"url": "https://example.com/very/long/url"}'
```

**Response:**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "url": "https://example.com/very/long/url",
  "shortCode": "xYz123",
  "createdAt": "2025-09-22T12:00:00Z",
  "updatedAt": "2025-09-22T12:00:00Z"
}
```

### Retrieve Original URL
```bash
curl http://localhost:8080/api/v1/shorten/xYz123
```

### Update Short URL
```bash
curl -X PUT http://localhost:8080/api/v1/shorten/xYz123 \
  -H "Content-Type: application/json" \
  -d '{"url": "https://example.com/updated/url"}'
```

### Get Statistics
```bash
curl http://localhost:8080/api/v1/shorten/xYz123/stats
```

**Response:**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "url": "https://example.com/very/long/url",
  "shortCode": "xYz123",
  "accessCount": 15,
  "createdAt": "2025-09-22T12:00:00Z",
  "updatedAt": "2025-09-22T12:00:00Z"
}
```

### Delete Short URL
```bash
curl -X DELETE http://localhost:8080/api/v1/shorten/xYz123
```

## ⚙️ Configuration

Configure the service using environment variables:

```env
# PocketBase
POCKETBASE_URL=http://localhost:8090

# Server
PORT=8080
ENVIRONMENT=development

# Short Code Generation
SHORT_CODE_LENGTH=6
MAX_RETRIES=5
```

## 🧪 Testing

Run all tests:
```bash
task test
```

Run tests with coverage:
```bash
task test:coverage
```

## 🔧 Development

### Available Task Commands

```bash
task                    # Show all available tasks
task build             # Build the application
task run               # Build and run the application
task dev               # Run with hot reload (auto-installs air)
task test              # Run tests
task test:coverage     # Run tests with coverage
task deps              # Download and tidy dependencies
task fmt               # Format code
task lint              # Run linter (auto-installs golangci-lint)
task check             # Run all code quality checks
task docker:build      # Build Docker image
task docker:run        # Run with Docker Compose
task db:up             # Start PocketBase container
task clean             # Clean build artifacts
task setup             # Initial project setup
```

### Hot Reload Development

For development with hot reloading:
```bash
task dev
```

This will automatically install Air if it's not present and start the development server with hot reload.

## 📊 Performance Considerations

- **Atomic Updates**: Access counts are updated atomically to prevent race conditions
- **Database Indexing**: Unique constraint on short codes for fast lookups
- **SQLite Performance**: Optimized SQLite backend with PocketBase
- **Async Operations**: Access count increments are performed asynchronously
- **UUID Primary Keys**: Using UUID v4 for distributed system compatibility

## 🔒 Security Features

- **Input Validation**: Comprehensive URL validation and sanitization
- **SQL Injection Protection**: PocketBase provides built-in protection
- **CORS Support**: Configurable Cross-Origin Resource Sharing
- **Error Handling**: Secure error messages that don't leak internal information
- **Rate Limiting**: Ready for rate limiting implementation (future enhancement)

## 🚀 Deployment

### Production Docker Build
```bash
task docker:build
```

### Azure App Service Deployment
The service is ready for deployment to Azure App Service with:
- Health check endpoint at `/health`
- Configurable port via `PORT` environment variable
- Production-ready logging and error handling

## 🛣️ Future Enhancements

- [ ] **Expiration Management**: TTL for URLs with automatic cleanup
- [ ] **Custom Short Codes**: Allow user-defined short codes
- [ ] **Advanced Analytics**: IP tracking, geographic data, referrer tracking
- [ ] **Authentication Integration**: Leverage PocketBase's built-in auth system
- [ ] **Real-time Dashboard**: WebSocket-powered live analytics using PocketBase subscriptions
- [ ] **Caching Layer**: Redis integration for improved performance
- [ ] **Rate Limiting**: Request throttling and API key authentication
- [ ] **Batch Operations**: Bulk URL shortening API
- [ ] **Web Interface**: Simple frontend for URL shortening
- [ ] **Metrics & Monitoring**: Prometheus metrics integration

## 📜 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🤝 Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📞 Support

For support and questions:
- Create an issue in the GitHub repository
- Check the [API documentation](docs/api.md) for detailed endpoint information
- Review the [troubleshooting guide](docs/troubleshooting.md) for common issues
