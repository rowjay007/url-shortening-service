# Go and PocketBase: Shortified!

Whether you're building your first web service or your hundredth microservice, the URL shortener remains a compelling case study. It's conceptually simple, quick to prototype, yet deceptively complex at scale.

Most tutorials stop at the basics: a single handler that accepts a URL, generates a random string, stores both in a database, and returns the short code. This works for demos, but what happens when that prototype becomes production-critical? When you need robust error handling, input validation, and graceful failure modes?

That gap between "works on my machine" and "scales in production" is where this tutorial begins.

We'll build a URL shortening service using Go and PocketBase, transforming a monolithic handler into a clean, layered architecture. You'll learn to structure Go services using clean architecture principles, implement structured error handling, design stable API contracts, and build resilient external integrations. The result: both a complete service and a blueprint for maintainable Go applications.

---

## The Monolithic Handler: A Lesson in Technical Debt

Version one prioritizes speed over structure. Using Gin's simplicity, we naturally gravitate toward putting everything in a single handler. The result: a function doing far too much work.

Consider this initial `createShortURL` handler:

```go
func createShortURL(c *gin.Context) {
    var request struct {
        URL string `json:"url"`
    }
    if err := c.BindJSON(&request); err != nil {
        c.JSON(400, gin.H{"error": "invalid request"})
        return
    }

    if !strings.HasPrefix(request.URL, "http") {
        c.JSON(400, gin.H{"error": "invalid URL"})
        return
    }

    shortCode := utils.GenerateRandomString(6)

    _, err := db.Exec("INSERT INTO urls (url, short_code) VALUES (?, ?)", request.URL, shortCode)
    if err != nil {
        log.Printf("db error: %v", err)
        c.JSON(500, gin.H{"error": "internal server error"})
        return
    }

    c.JSON(200, gin.H{"short_code": shortCode})
}
```

This code works but creates insurmountable technical debt. The engineering failures:

1.  **Untestable:** Unit testing requires mocking the entire Gin context and database layer—or maintaining a live database. Testing business logic in isolation becomes impossible.

2.  **Violates Single Responsibility:** This function handles HTTP decoding, input validation, ID generation, database interaction, and HTTP encoding. Five distinct responsibilities in one place.

3.  **Framework Coupling:** Business logic is inseparable from Gin. Exposing the same functionality via gRPC or CLI requires complete rewrites.

4.  **Opaque Failures:** Generic 500 errors provide no diagnostic information. Database failures, constraint violations, and application bugs all look identical.

This technical debt compounds rapidly. Establishing clear boundaries isn't complexity—it's the foundation for sustainable growth.

### The Power of Delegation

The clean architecture version delegates each responsibility to specialized components:

```go
func (h *URLHandler) CreateShortURL(c *gin.Context) {
    var req dto.CreateURLRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        h.handleServiceError(c, serviceErrors.NewBadRequestError("handler.CreateShortURL", "invalid request body", err))
        return
    }

    response, err := h.service.CreateShortURL(c.Request.Context(), &req)
    if err != nil {
        h.handleServiceError(c, err)
        return
    }

    c.JSON(http.StatusCreated, response)
}
```

This handler has exactly three responsibilities: decode requests, delegate to services, and encode responses. No business logic, validation, or database calls.

**The transformation yields immediate benefits:**

*   **Testability:** Mock the `URLService` interface for isolated unit tests
*   **Maintainability:** Business logic changes never touch handlers
*   **Clarity:** New engineers understand the flow without implementation details

This delegation pattern forms the foundation of our layered architecture.

---

## Layered Architecture: Enforcing Separation of Concerns

Clean Architecture solves the monolith problem through strict separation of concerns. The **Dependency Rule** governs everything: dependencies point inward toward business logic. Outer layers know inner layers, but never the reverse.

Our project structure reflects these layers:

```
internal/
├── handlers/        # Layer 4: The Web Framework (Gin)
├── services/        # Layer 3: The Business Logic
├── repository/      # Layer 2: The Data Access Abstraction
└── models/          # Layer 1: The Core Domain Entities

// Supporting Packages
├── dto/             # API Contracts (Request/Response Structs)
├── errors/          # Custom Error Types
└── validator/       # Input Validation Logic
```

Go's interfaces enforce the **Dependency Inversion Principle**. High-level modules depend on abstractions, not implementations.

**The Repository Interface defines *what*, not *how*:**

```go
type URLRepository interface {
    Create(ctx context.Context, shortURL *models.ShortURL) error
    GetByShortCode(ctx context.Context, shortCode string) (*models.ShortURL, error)
    ExistsByShortCode(ctx context.Context, shortCode string) (bool, error)
    UpdateAccessCount(ctx context.Context, shortCode string) error
}
```

**The Service Interface encapsulates business capabilities:**

```go
type URLService interface {
    CreateShortURL(ctx context.Context, req *dto.CreateURLRequest) (*dto.CreateURLResponse, error)
    GetOriginalURL(ctx context.Context, shortCode string) (*dto.GetURLResponse, error)
    GetStats(ctx context.Context, shortCode string) (*dto.GetStatsResponse, error)
}
```

**Dependency Injection in `main.go`:**
The composition root wires concrete implementations to interfaces:

```go
func main() {
    config := loadConfig()
    pocketBaseClient := database.NewPBClient(config.PocketBaseURL)
    
    urlRepo := repository.NewURLRepository(pocketBaseClient)
    urlService := services.NewURLService(urlRepo, validator.NewURLValidator())
    urlHandler := handlers.NewURLHandler(urlService)

    router := gin.Default()
    router.POST("/api/v1/shorten", urlHandler.CreateShortURL)
    router.GET("/api/v1/:code", urlHandler.GetOriginalURL)
    router.GET("/api/v1/:code/stats", urlHandler.GetStats)
    
    router.Run(":8080")
}
```

Each layer tests in isolation by mocking dependencies. Tight coupling is eliminated.

Component replacement becomes trivial—swap PocketBase for PostgreSQL by implementing `URLRepository`. Service and handler layers remain untouched.

### Request Lifecycle: Layer Collaboration in Action

A `POST` to `/api/v1/shorten` flows through three distinct layers:

**Handler Layer:**
- Decodes JSON to `dto.CreateURLRequest`
- Delegates to service layer
- Translates service errors to HTTP responses
- Returns `201` with `dto.CreateURLResponse`

**Service Layer:**
- Validates input via `validator` package
- Checks for duplicate custom codes
- Generates unique codes when needed
- Orchestrates repository calls
- Maps domain models to DTOs

**Repository Layer:**
- Translates domain models to PocketBase schema
- Executes HTTP requests with context propagation
- Interprets status codes (409 → duplicate error)
- Updates domain models with generated IDs

Each layer communicates through well-defined contracts, enabling independent testing and modification.

---

## API Contracts: Decoupling External Interfaces from Internal Models

Robust APIs separate internal domain models from external contracts. This discipline prevents API brittleness and information leakage.

### The Anti-Pattern: Direct Model Exposure

Using domain models directly in handlers seems efficient but creates brittle API-to-database coupling:

```go
func (h *URLHandler) CreateShortURL(c *gin.Context) {
    var url models.ShortURL
    if err := c.ShouldBindJSON(&url); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
        return
    }
    
    c.JSON(http.StatusCreated, url)
}
```

**The problems compound quickly:**

1.  **Breaking Changes:** Database schema changes break public APIs
2.  **Information Leakage:** Internal fields, flags, and metadata leak to clients
3.  **Inflexibility:** Database-optimized structures rarely match API requirements

### Data Transfer Objects: Explicit API Contracts

**Data Transfer Objects** create explicit API contracts, decoupling external interfaces from internal models:

**Domain Model** (internal source of truth):

```go
package models

import "time"

type ShortURL struct {
    ID          string    `db:"id"`
    URL         string    `db:"url"`
    ShortCode   string    `db:"short_code"`
    AccessCount int64     `db:"access_count"`
    Created     time.Time `db:"created"`
    Updated     time.Time `db:"updated"`
}
```

**API Contract** (external interface):

```go
package dto

import "time"

type CreateURLRequest struct {
    URL        string  `json:"url" validate:"required,url,max=2048"`
    CustomCode *string `json:"customCode,omitempty" validate:"omitempty,min=4,max=20,alphanum"`
}

type CreateURLResponse struct {
    ID        string    `json:"id"`
    URL       string    `json:"url"`
    ShortCode string    `json:"shortCode"`
    CreatedAt time.Time `json:"createdAt"`
}

type GetStatsResponse struct {
    ShortCode   string    `json:"shortCode"`
    AccessCount int64     `json:"accessCount"`
    CreatedAt   time.Time `json:"createdAt"`
}
```

The service layer mediates between domains, mapping DTOs to models and vice versa. This translation enables independent evolution of APIs and database schemas.

### Explicit Mappers: Scaling the Pattern

Dedicated mapper functions clarify transformation logic:

```go
func toCreateURLResponse(url *models.ShortURL) *dto.CreateURLResponse {
    return &dto.CreateURLResponse{
        ID:        url.ID,
        URL:       url.URL,
        ShortCode: url.ShortCode,
        CreatedAt: url.Created,
    }
}

func fromCreateURLRequest(req *dto.CreateURLRequest) (*models.ShortURL, error) {
    return &models.ShortURL{
        URL:       req.URL,
        ShortCode: "",
    }, nil
}
```

Mappers make service intentions explicit:

```go
func (s *urlServiceImpl) CreateShortURL(ctx context.Context, req *dto.CreateURLRequest) (*dto.CreateURLResponse, error) {
    if err := s.validator.ValidateURL(req.URL); err != nil {
        return nil, err
    }

    shortURL, err := fromCreateURLRequest(req)
    if err != nil {
        return nil, serviceErrors.NewInternalError("service.Create", "failed to map request", err)
    }

    shortURL.ShortCode = s.generateUniqueShortCode(ctx)

    if err := s.repo.Create(ctx, shortURL); err != nil {
        return nil, err
    }

    return toCreateURLResponse(shortURL), nil
}
```

This discipline maintains clean boundaries: DTOs define external language, models define internal language, services translate between them.

---

## Structured Error Handling: Beyond Generic 500s

Error handling quality distinguishes robust systems from fragile ones. Generic errors provide no actionable information.

### The Anti-Pattern: String-Based Error Handling

Generic errors destroy diagnostic capability:

```go
func (s *urlServiceImpl) GetOriginalURL(ctx context.Context, shortCode string) (string, error) {
    url, err := s.repo.GetByShortCode(ctx, shortCode)
    if err != nil {
        if err == sql.ErrNoRows {
            return "", fmt.Errorf("not found")
        }
        return "", fmt.Errorf("internal error")
    }
    return url.URL, nil
}

_, err := h.service.GetOriginalURL(c.Request.Context(), shortCode)
if err != nil {
    if err.Error() == "not found" {
        c.JSON(http.StatusNotFound, gin.H{"error": "URL not found"})
    } else {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "An internal error occurred"})
    }
    return
}
```

**Three critical failures:**

1.  **Context Loss:** Original database errors disappear
2.  **Brittle Coupling:** String comparisons break when messages change
3.  **Diagnostic Poverty:** "Internal error" provides no actionable information

### Structured Error Types as Domain Objects

Structured errors become first-class domain objects, carrying rich diagnostic information:

```go
package errors

type ErrorCode int

const (
    ErrorCodeNotFound ErrorCode = iota + 1
    ErrorCodeDuplicate
    ErrorCodeValidation
    ErrorCodeInternal
)

type ServiceError struct {
    Op      string    // Operation that failed
    Code    ErrorCode // Machine-readable type
    Message string    // Human-readable message
    Err     error     // Wrapped original error
}

func (e *ServiceError) Error() string {
    return fmt.Sprintf("%s: %s", e.Op, e.Message)
}

func NewNotFoundError(op, message string) *ServiceError {
    return &ServiceError{Op: op, Code: ErrorCodeNotFound, Message: message}
}
```

**Structured errors provide four key capabilities:**

1.  **Context:** Operation-specific failure points
2.  **Classification:** Machine-readable error categories
3.  **Clarity:** User-appropriate messages
4.  **Traceability:** Wrapped original errors for debugging

### Centralized Error Translation

A single handler method translates structured errors to HTTP responses:

```go
func (h *URLHandler) handleServiceError(c *gin.Context, err error) {
    var serviceErr *serviceErrors.ServiceError
    if errors.As(err, &serviceErr) {
        statusCode := map[serviceErrors.ErrorCode]int{
            serviceErrors.ErrorCodeNotFound:   http.StatusNotFound,
            serviceErrors.ErrorCodeDuplicate:  http.StatusConflict,
            serviceErrors.ErrorCodeValidation: http.StatusBadRequest,
        }[serviceErr.Code]
        if statusCode == 0 {
            statusCode = http.StatusInternalServerError
        }

        log.Error().Err(serviceErr.Err).Str("op", serviceErr.Op).Msg(serviceErr.Message)
        c.JSON(statusCode, dto.ErrorResponse{Error: serviceErr.Message})
        return
    }

    log.Error().Err(err).Msg("Unexpected error")
    c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "Internal server error"})
}
```

This centralized approach ensures consistent error responses while logging rich diagnostic information without leaking implementation details.

### Structured Logging for Observability

Structured logging produces machine-readable diagnostic data:

```json
{
  "level": "error",
  "op": "repository.GetByShortCode",
  "error": "record not found",
  "message": "short URL not found",
  "time": "2023-10-27T10:00:00Z"
}
```

This enables powerful operational capabilities: filtering by operation, counting error types, and automated alerting. The transformation from reactive debugging to proactive observability.

---

## Input Validation: The Gatekeeper Pattern

Defensive programming rejects invalid data at system boundaries. Scattered validation creates security vulnerabilities and behavioral inconsistencies.

### The Anti-Pattern: Distributed Validation Logic

Scattered validation mixes concerns and duplicates logic:

```go
func (s *urlServiceImpl) CreateShortURL(ctx context.Context, req *dto.CreateURLRequest) (*dto.CreateURLResponse, error) {
    if len(req.URL) > 2048 {
        return nil, serviceErrors.NewValidationError("service.Create", "URL exceeds maximum length", nil)
    }
    if !strings.HasPrefix(req.URL, "http") {
        return nil, serviceErrors.NewValidationError("service.Create", "Invalid URL format", nil)
    }
    
    // Business logic mixed with validation
    shortCode, err := s.generateUniqueShortCode(ctx)
    // ...
}
```

**The maintenance problems:**

1.  **Code Duplication:** Multiple methods replicate identical validation rules
2.  **Mixed Concerns:** Business logic contaminated with input validation
3.  **Inconsistent Rules:** No single source of truth for validation logic

### Centralized Validation as System Gatekeeper

The `validator` package creates a dedicated gatekeeper for external input:

**Core Principles:**
- Pure functions with no side effects
- No business logic knowledge
- Single source of truth for validation rules

```go
package validator

import (
    "net/url"
    "strings"
    "github.com/rowjay/url-shortening-service/internal/constants"
    "github.com/rowjay/url-shortening-service/internal/errors"
)

type URLValidator struct {
    blockedDomains []string
}

func NewURLValidator() *URLValidator {
    return &URLValidator{blockedDomains: constants.BlockedDomains}
}

func (v *URLValidator) ValidateURL(rawURL string) error {
    if len(rawURL) > constants.MaxURLLength {
        return errors.NewValidationError("validator.ValidateURL", "URL exceeds maximum length", nil)
    }

    parsed, err := url.ParseRequestURI(rawURL)
    if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
        return errors.NewValidationError("validator.ValidateURL", "Invalid URL format", err)
    }

    for _, domain := range v.blockedDomains {
        if strings.Contains(parsed.Host, domain) {
            return errors.NewValidationError("validator.ValidateURL", "URL domain is blocked", nil)
        }
    }

    return nil
}
```

Services consume validators before executing business logic:

```go
func (s *urlServiceImpl) CreateShortURL(ctx context.Context, req *dto.CreateURLRequest) (*dto.CreateURLResponse, error) {
    if err := s.validator.ValidateURL(req.URL); err != nil {
        return nil, err
    }
    if req.CustomCode != nil {
        if err := s.validator.ValidateShortCode(*req.CustomCode); err != nil {
            return nil, err
        }
    }

    shortCode := req.CustomCode
    if shortCode == nil {
        code, err := s.generateUniqueShortCode(ctx)
        if err != nil {
            return nil, err
        }
        shortCode = &code
    }

    shortURL := &models.ShortURL{
        URL:       req.URL,
        ShortCode: *shortCode,
    }

    if err := s.repo.Create(ctx, shortURL); err != nil {
        return nil, err
    }

    return &dto.CreateURLResponse{
        ID:        shortURL.ID,
        URL:       shortURL.URL,
        ShortCode: shortURL.ShortCode,
        CreatedAt: shortURL.Created,
    }, nil
}
```

Centralized validation ensures consistency and maintainability. Rule changes occur in a single location, enabling services to focus exclusively on business logic with pre-validated inputs.

### Multi-Layer Validation Strategy

**Five validation categories ensure comprehensive input security:**

1.  **Syntactical:** RFC 3986 compliance via `url.ParseRequestURI`
2.  **Protocol:** HTTP/HTTPS scheme enforcement
3.  **Domain Blocking:** Configurable blocklist for malicious sites
4.  **Length Constraints:** DoS prevention through size limits
5.  **Character Sets:** Alphanumeric enforcement prevents injection attacks

This comprehensive approach creates a security-first input boundary while maintaining system auditability.

---

## External Integration: The Repository as Adapter

The repository layer bridges domain models and external persistence. Poor implementation creates leaky abstractions that violate architectural boundaries.

### The Anti-Pattern: Leaky Abstractions

Repositories that expose implementation details contaminate business logic:

```go
func (s *urlServiceImpl) CreateShortURL(ctx context.Context, req *dto.CreateURLRequest) (*dto.CreateURLResponse, error) {
    shortURL := &models.ShortURL{
        URL:       req.URL,
        ShortCode: req.CustomCode,
    }

    pbReq := map[string]interface{}{
        "URL": shortURL.URL,
        "ShortCode": shortURL.ShortCode,
    }

    resp, err := s.repo.Create(ctx, pbReq)
    if err != nil {
        return nil, err
    }

    if resp.StatusCode == http.StatusConflict {
        return nil, serviceErrors.NewDuplicateError("service.Create", "short code exists")
    }
    
    return toCreateURLResponse(shortURL), nil
}
```

**Architectural violations:**

1.  **Dependency Rule Violation:** Services become coupled to HTTP implementation details
2.  **Testing Complexity:** Requires mocking HTTP responses instead of domain interfaces
3.  **Replacement Difficulty:** Technology changes force service layer rewrites

### Pure Adapter Implementation

True adapters translate between domain interfaces and external protocols without leaking implementation details.

**Architectural benefits of treating PocketBase as a remote service:**

1.  **Complete Decoupling:** Technology replacement affects only repository implementations
2.  **Feature Leverage:** Access to authentication, validation, and real-time capabilities
3.  **Forced Abstraction:** Network boundaries prevent architectural violations

Implementation demonstrates clean domain-to-HTTP translation:

```go
type urlRepositoryImpl struct {
    pb *database.PBClient
}

func (r *urlRepositoryImpl) Create(ctx context.Context, shortURL *models.ShortURL) error {
    reqBody := pocketBaseCreateRequest{
        URL:         shortURL.URL,
        ShortCode:   shortURL.ShortCode,
        AccessCount: 0,
    }

    jsonBody, err := json.Marshal(reqBody)
    if err != nil {
        return serviceErrors.NewInternalError("repository.Create", "failed to marshal request", err)
    }

    endpoint := fmt.Sprintf("%s/api/collections/%s/records", r.pb.BaseURL, constants.ShortURLsCollection)
    req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(jsonBody))
    if err != nil {
        return serviceErrors.NewInternalError("repository.Create", "failed to create request", err)
    }
    req.Header.Set("Content-Type", "application/json")

    resp, err := r.pb.HTTPClient.Do(req)
    if err != nil {
        return serviceErrors.NewInternalError("repository.Create", "failed to execute request", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusCreated {
        if resp.StatusCode == http.StatusConflict {
            return serviceErrors.NewDuplicateError("repository.Create", "short code already exists")
        }
        return serviceErrors.NewInternalError("repository.Create", "PocketBase returned an error", fmt.Errorf("status: %d", resp.StatusCode))
    }

    var pbResp pocketBaseResponse
    if err := json.NewDecoder(resp.Body).Decode(&pbResp); err != nil {
        return serviceErrors.NewInternalError("repository.Create", "failed to decode response", err)
    }

    shortURL.ID = pbResp.ID
    shortURL.Created = pbResp.Created
    shortURL.Updated = pbResp.Updated
    
    return nil
}
```

This repository exemplifies the **Adapter Pattern**: translating domain interfaces to external protocols while maintaining clean architectural boundaries.

---

## Context Propagation: Request Lifecycle Management

`context.Context` distinguishes professional Go services from amateur implementations. Without proper context handling, services suffer resource leaks, zombie goroutines, and cascading failures under load.

**The problem:** User cancellations and timeouts don't propagate to downstream operations, causing resource waste and system instability.

**The solution:** Context carries cancellation signals, deadlines, and request-scoped values across service boundaries.

### Three-Layer Context Strategy

**Context flows through architectural layers:**

1.  **Handler Layer:** Gin provides request contexts with automatic client disconnect handling
2.  **Service Layer:** Pure passthrough—no context inspection or modification
3.  **Repository Layer:** Context consumption for timeout-aware external calls

Repository implementation demonstrates context utilization:

```go
func (r *urlRepositoryImpl) GetByShortCode(ctx context.Context, shortCode string) (*models.ShortURL, error) {
    ctx, cancel := context.WithTimeout(ctx, constants.RequestTimeout)
    defer cancel()

    filter := fmt.Sprintf("short_code='%s'", shortCode)
    reqURL := fmt.Sprintf("%s/api/collections/%s/records?filter=%s", r.pb.BaseURL, constants.ShortURLsCollection, url.QueryEscape(filter))

    req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
    if err != nil {
        return nil, serviceErrors.NewInternalError("repository.GetByShortCode", "failed to create request", err)
    }

    resp, err := r.pb.HTTPClient.Do(req)
    if err != nil {
        return nil, serviceErrors.NewInternalError("repository.GetByShortCode", "failed to execute request", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode == http.StatusNotFound {
        return nil, serviceErrors.NewNotFoundError("repository.GetByShortCode", "short URL not found")
    }
    if resp.StatusCode != http.StatusOK {
        return nil, serviceErrors.NewInternalError("repository.GetByShortCode", "PocketBase returned an error", fmt.Errorf("status: %d", resp.StatusCode))
    }

    var pbResp struct {
        Items []pocketBaseShortURL `json:"items"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&pbResp); err != nil {
        return nil, serviceErrors.NewInternalError("repository.GetByShortCode", "failed to decode response", err)
    }

    if len(pbResp.Items) == 0 {
        return nil, serviceErrors.NewNotFoundError("repository.GetByShortCode", "short URL not found")
    }

    item := pbResp.Items[0]
    return &models.ShortURL{
        ID:          item.ID,
        URL:         item.URL,
        ShortCode:   item.ShortCode,
        AccessCount: item.AccessCount,
        Created:     item.Created,
        Updated:     item.Updated,
    }, nil
}
```

**Context provides two critical guarantees:**

-   **Timeout Protection:** 30-second maximum prevents cascading failures from slow dependencies
-   **Cancellation Propagation:** Client disconnects immediately abort downstream operations

Disciplined context usage forms the foundation of resilient Go services.

---

## Architectural Principles: A Practical Blueprint

We transformed a monolithic handler into a layered architecture through deliberate engineering decisions. These weren't premature optimizations but pragmatic choices enabling maintainability, testability, and resilience.

**Six principles for sustainable Go services:**

1.  **Domain Isolation:** Business logic depends only on interfaces, never implementations
2.  **Interface Contracts:** Define layer boundaries with interfaces for testability and modularity
3.  **DTO Separation:** Decouple public APIs from internal models to prevent breaking changes
4.  **Structured Errors:** Rich error types enable consistent handling and observability
5.  **Edge Validation:** Centralized input validation creates secure system boundaries
6.  **Context Propagation:** Request lifecycle management prevents resource leaks and cascading failures

These principles transform code into engineered systems—organized workshops where any engineer contributes confidently, not chaotic garages requiring tribal knowledge. The architectural investment pays dividends through reduced bugs, accelerated development, and sustainable codebases.

---

## Conclusion

We began with a familiar scenario: a monolithic handler that worked but couldn't scale. Through systematic application of clean architecture principles, we transformed it into a resilient, maintainable service. This transformation wasn't about adding complexity—it was about organizing complexity.

The URL shortener now demonstrates six critical patterns that apply to any Go service: domain isolation through service layers, interface-based contracts for testability, explicit API boundaries via DTOs, structured error handling for observability, centralized input validation for security, and context-aware operations for resilience.

Apply these patterns incrementally to existing services. Start with error handling and input validation—they provide immediate value with minimal refactoring. Then introduce service interfaces and DTOs to create testable boundaries. The repository pattern comes last, as it requires the most architectural change but provides the greatest long-term flexibility.

The goal isn't perfection from day one. It's building systems that can evolve safely, scale predictably, and welcome new contributors without archaeological expeditions through legacy code.

---

## Project Resources

- **GitHub Repository:** [github.com/rowjay007/url-shortening-service](https://github.com/rowjay007/url-shortening-service)
- **Original Project Brief:** [roadmap.sh/projects/url-shortening-service](https://roadmap.sh/projects/url-shortening-service)
- **Author:** [@rowjay](https://github.com/rowjay007)
