# Go and PocketBase: Shortified!

Whether you're building your first web service or your hundredth microservice, the URL shortener remains a compelling case study. It's conceptually simple, quick to prototype, yet deceptively complex at scale.

A single handler that accepts a URL, generates a random string, stores both in a database, and returns the short code. This works for demos, but what happens when that prototype becomes production-critical? When you need robust error handling, input validation, and graceful failure modes?

That gap between "works on my machine" and "scales in production" is where our journey begins.

Together, we'll build a URL shortening service using Go and PocketBase, transforming a monolithic handler into a clean, layered architecture. I'll show you how to structure Go services using clean architecture principles, implement structured error handling, design stable API contracts, and build resilient external integrations. By the end, you'll have both a complete service and a blueprint for your next maintainable Go application.

---

## The Monolithic Handler: A Lesson in Technical Debt

Let's start with the honest reality of version one. Our goal is to get a functional endpoint up and running as quickly as possible. Using Gin's simplicity, we naturally gravitate toward putting everything in a single handler. The result: a function doing far too much work.

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

This code works, and we ship it. But we've just created insurmountable technical debt. Let's examine the specific engineering failures we've introduced:

1.  **Untestable:** Unit testing requires mocking the entire Gin context and database layer—or maintaining a live database. Testing business logic in isolation becomes impossible.

2.  **Violates Single Responsibility:** This function handles HTTP decoding, input validation, ID generation, database interaction, and HTTP encoding. Five distinct responsibilities in one place.

3.  **Framework Coupling:** Business logic is inseparable from Gin. Exposing the same functionality via gRPC or CLI requires complete rewrites.

4.  **Opaque Failures:** Generic 500 errors provide no diagnostic information. Database failures, constraint violations, and application bugs all look identical.

This is the technical debt that will cripple our project. The first step to paying it down is establishing clear boundaries. This isn't about adding unnecessary complexity—it's about creating a structure that enables future growth and maintainability.

### The Power of Delegation

Now, let's look at how we can build that same handler using a clean architecture approach. Notice what it *doesn't* do:

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

This is a night-and-day difference. Our new handler is lean, focused, and incredibly simple. Its only responsibilities are to decode the request, call the service, and encode the response. It contains no business logic, no validation, and no database calls.

**The benefits are immediate:**

*   **Testability:** We can easily test this handler by providing a mock `URLService`. We don't need a database or complex setup. We can verify its behavior in complete isolation.
*   **Maintainability:** If we need to change how short codes are generated, we don't touch the handler. If we need to add a new validation rule, we don't touch the handler. Its purpose is stable and unchanging.
*   **Clarity:** A new engineer can look at this and immediately grasp the flow of control without getting bogged down in implementation details.

By adopting this clean, delegated approach, we've already made our system more robust and maintainable. Now, let's explore the architecture that makes this clean separation possible.

---

## Layered Architecture: Enforcing Separation of Concerns

To solve the problems of our prototype, we need to enforce a strict separation of concerns. We'll adopt a layered architecture inspired by the principles of Clean Architecture. This isn't an academic exercise—it's a pragmatic approach to building resilient software.

The core principle is the **Dependency Rule**: all dependencies must point inwards, toward the core business logic. The outer layers know about the inner layers, but the inner layers know nothing about the outer ones.

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

This structure is enforced using Go's interfaces, which allow us to implement the **Dependency Inversion Principle**. Instead of high-level modules depending on low-level modules, both depend on abstractions (interfaces).

**The Repository Interface (The Data Contract):**
This interface defines *what* we can do with our data, but not *how*. It's the contract between our business logic and the persistence layer.

```go
type URLRepository interface {
    Create(ctx context.Context, shortURL *models.ShortURL) error
    GetByShortCode(ctx context.Context, shortCode string) (*models.ShortURL, error)
    ExistsByShortCode(ctx context.Context, shortCode string) (bool, error)
    UpdateAccessCount(ctx context.Context, shortCode string) error
}
```

**The Service Interface (The Business Logic Contract):**
This defines the core capabilities of our application, completely independent of any web framework.

```go
type URLService interface {
    CreateShortURL(ctx context.Context, req *dto.CreateURLRequest) (*dto.CreateURLResponse, error)
    GetOriginalURL(ctx context.Context, shortCode string) (*dto.GetURLResponse, error)
    GetStats(ctx context.Context, shortCode string) (*dto.GetStatsResponse, error)
}
```

**Wiring It All Together (Dependency Injection):**
Our `main.go` function becomes the **composition root**. It's the only place in the application that knows about the concrete implementations of these interfaces. It builds the dependency graph and injects the concrete types.

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

This is a paradigm shift. Each layer can now be tested in isolation by mocking its dependencies—the service layer mocks the repository, the handler layer mocks the service. We have broken the tight coupling that made our prototype so fragile.

This architecture allows us to swap out components with minimal impact. If we decide to move from PocketBase's REST API to a direct SQLite driver or even to a different database like PostgreSQL, we only need to write a new implementation of the `URLRepository` interface. The service and handler layers remain completely untouched. This is the definition of a maintainable and scalable system.

### The Flow of a Request: A Practical Walkthrough

Let's trace the lifecycle of a single API call to see how our layers work together. A user wants to create a new short URL by sending a `POST` request to `/api/v1/shorten`.

1.  **The Handler (`handlers/url_handler.go`):**
    *   The Gin router receives the incoming HTTP request and routes it to our `CreateShortURL` handler.
    *   The handler's first and only job is to manage the HTTP interaction. It uses `c.ShouldBindJSON()` to decode the JSON payload into a `dto.CreateURLRequest` struct.
    *   If binding fails, the handler immediately returns a `400 Bad Request` response. It knows nothing about *why* the binding failed, only that the request was malformed.
    *   If binding is successful, the handler calls the service layer: `h.service.CreateShortURL(ctx, &req)`.
    *   The handler then waits for the service to return either a `dto.CreateURLResponse` or an `error`.
    *   If an error is returned, it passes the error to the `handleServiceError` function to be translated into the correct HTTP status code and response body.
    *   If a response is returned, it serializes the DTO to JSON and sends it back to the client with a `201 Created` status code.

2.  **The Service (`services/url_service.go`):**
    *   The `CreateShortURL` method in the service receives the `dto.CreateURLRequest`.
    *   Its first action is to perform validation by calling the `validator.ValidateURL()` and `validator.ValidateShortCode()` methods. If validation fails, it returns a `ServiceError` with the code `ErrorCodeValidation`.
    *   Next, it orchestrates the business logic. If the user provided a custom code, it calls `repo.ExistsByShortCode()`. If the code exists, it returns a `ServiceError` with the code `ErrorCodeDuplicate`.
    *   If no custom code was provided, it enters a loop to generate a unique short code, calling `utils.GenerateShortCode()` and `repo.ExistsByShortCode()` until a unique code is found.
    *   Once it has a valid URL and a unique short code, it constructs a `models.ShortURL` domain object.
    *   It then calls `repo.Create(ctx, &shortURL)` to persist the new record.
    *   Finally, it maps the resulting `models.ShortURL` (now populated with an ID and timestamps from the database) to a `dto.CreateURLResponse` and returns it to the handler.

3.  **The Repository (`repository/url_repository.go`):**
    *   The `Create` method receives the `models.ShortURL` object.
    *   It maps this domain model to a `pocketBaseCreateRequest` struct, which matches the schema expected by the PocketBase API.
    *   It constructs an `http.Request` with the appropriate method, URL, and body, ensuring the `context` is passed along for cancellation and timeouts.
    *   It executes the HTTP request against the PocketBase server.
    *   It then interprets the HTTP response. A `201 Created` is a success. A `409 Conflict` is translated into our `serviceErrors.NewDuplicateError`. Any other non-2xx status is translated into a `serviceErrors.NewInternalError`.
    *   On success, it decodes the response body from PocketBase to get the database-generated ID and timestamps, and updates the original `models.ShortURL` object with this information before returning.

This clear, unidirectional flow is the hallmark of a well-architected system. Each layer has a specific job, and it communicates with the layers adjacent to it through well-defined contracts (interfaces and DTOs). This makes the system easy to reason about, debug, and extend.

---

## Defining the Lines: Stable API Contracts with DTOs

A critical discipline in building robust APIs is the strict separation of your internal data structures (domain models) from your external data structures (the API contract). Let's look at the common anti-pattern first.

### The Anti-Pattern: Exposing Your Database Models Directly

In a rush to get things working, it's incredibly tempting to use your internal `models.ShortURL` struct directly in your handler for both request binding and response serialization.

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

This shortcut seems efficient, but it creates a tight, brittle coupling between your public API and your private database schema. This leads to several serious problems:

**The Problem with Direct Exposure:**

1.  **Breaking Changes:** If you rename a database column, you've just introduced a breaking change to your public API. Your API becomes a fragile mirror of your database schema.
2.  **Information Leakage:** Your internal models may contain fields that are irrelevant or sensitive and should not be exposed to the client, such as internal flags, metadata, or hashed values.
3.  **Use-Case Mismatch:** The data structure that's optimal for your database is rarely the optimal structure for a specific API response. A `GET` request might need a subset of fields, while a `stats` endpoint might need aggregated data.

### The Solution: Defining Explicit API Contracts with DTOs

To solve these problems, you must treat your API as a public, versioned contract. The best way to do this is with **Data Transfer Objects (DTOs)**. These are simple, plain structs whose sole purpose is to define the exact shape of your API's requests and responses. They live in their own `dto/` package and are the public face of your service.

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

The **service layer** acts as the mediator, responsible for mapping between these two worlds. It takes an incoming `CreateURLRequest` DTO, validates it, and then maps its data into a `models.ShortURL` domain object to be sent to the repository. Conversely, when fetching data, it takes a `models.ShortURL` from the repository and maps it to a `CreateURLResponse` or `GetStatsResponse` DTO before returning it to the handler.

This mapping is a small price to pay for immense flexibility. It decouples our API contract from our database schema, allowing each to evolve independently. It's a crucial step in building an API that is stable, secure, and designed for its consumers.

### The Mapper Pattern: A Clean Implementation

To keep our service layer clean, we can even introduce explicit mapper functions. While not strictly necessary for a project of this size, it's a pattern that scales well. These mappers can live in the service layer or even their own package.

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

By using mappers, the service layer's intent becomes crystal clear:

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

This level of discipline ensures that the boundaries between layers remain pristine. The DTOs define the public-facing language of our API, while the domain models define the internal language of our business logic. The service layer acts as the professional translator between the two.

---

## Beyond 500: A Strategy for Meaningful Error Handling

One of the most telling signs of a fragile system is its error handling. Let's examine the common anti-pattern that leads to opaque, unhelpful APIs.

### The Anti-Pattern: Generic Errors and String Comparisons

In a simple implementation, it's common to see error handling like this:

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

This approach is a debugging nightmare for several reasons:

1.  **Loss of Context:** The original error from the database (`sql.ErrNoRows`) is discarded. We lose the valuable context of *what* exactly went wrong.
2.  **Fragile String Comparisons:** Relying on `err.Error() == "not found"` is incredibly brittle. If a developer changes the error message in the service, the handler's logic breaks. This is a common source of bugs.
3.  **Ambiguity:** A generic "internal error" tells the client and our operations team nothing. Is the database down? Is there a bug in our query? We have no way to know without digging through logs.

### The Solution: Errors as First-Class, Structured Citizens

To fix this, you must elevate errors to be a core, designed part of your system. Instead of passing around simple strings, we'll create a dedicated `errors/` package to define a structured, custom error type that will be used throughout the application.

**The `ServiceError` Struct:**

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

With this structure in place, our repository and service layers can now return rich, meaningful errors. Instead of `fmt.Errorf("not found")`, our repository now returns `serviceErrors.NewNotFoundError("repository.GetByShortCode", "short URL not found")`.

This provides several key advantages:

1.  **Context:** We know the exact operation (`Op`) that failed.
2.  **Classification:** We have a machine-readable `Code` that allows us to programmatically handle different error types.
3.  **Clarity:** We have a clear `Message` intended for the end-user.
4.  **Traceability:** We can wrap the original low-level error (`Err`) for detailed logging without exposing it to the client.

**Centralized Error Handling in the Handler:**
The true power of this pattern is realized in the handler layer, where we can create a single, centralized function to translate any `ServiceError` into the correct HTTP response.

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

This function is a game-changer. It ensures that all API error responses are consistent and predictable. It guarantees that we log rich, contextual information for debugging while never leaking implementation details to the client. This systematic approach to error handling transformed our service from brittle to resilient.

### The Power of Structured Logging

Notice the logging line within our `handleServiceError` function:

```go
log.Error().Err(serviceErr.Err).Str("op", serviceErr.Op).Msg(serviceErr.Message)
```

This is not an accident. We are using the `zerolog` library to produce structured, JSON-formatted logs. When this error is logged, it won't be a simple, unparseable string. It will be a rich, machine-readable JSON object:

```json
{
  "level": "error",
  "op": "repository.GetByShortCode",
  "error": "record not found",
  "message": "short URL not found",
  "time": "2023-10-27T10:00:00Z"
}
```

This is a massive operational advantage. We can now ship these logs to a centralized logging platform (like Elasticsearch, Datadog, or Logz.io) and perform powerful queries. We can easily filter for all errors originating from a specific operation (`op`), count the occurrences of different error codes, and set up automated alerts for spikes in internal server errors. This is the difference between reactive debugging (grepping through raw text files) and proactive observability.

By treating errors as structured data, both in our API responses and in our internal logs, we build a system that is not only more reliable but also far easier to monitor and maintain at scale.

---

## First Line of Defense: Proactive and Centralized Validation

Defensive programming is key to building secure and reliable systems. The best way to handle invalid data is to reject it at the earliest possible moment. Let's examine the common anti-pattern that leads to security holes and inconsistent behavior.

### The Anti-Pattern: Scattered and Inconsistent Validation

When you're moving quickly, it's tempting to sprinkle validation checks directly within your service or handler logic wherever they seem to be needed.

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

This approach quickly becomes a maintenance nightmare:

1.  **Violation of DRY (Don't Repeat Yourself):** If multiple service methods need to validate a URL, you'll end up copying and pasting the same validation logic, leading to inconsistencies when one is updated and the other is forgotten.
2.  **Mixing Concerns:** The service layer's job is to orchestrate business logic, not to be an expert in the minutiae of what constitutes a valid URL string. This mixing of responsibilities makes the code harder to read and reason about.
3.  **Inconsistent Rules:** Without a central authority for validation, it's easy for different parts of the application to end up with slightly different rules, leading to unpredictable behavior for your users.

### The Solution: The Gatekeeper Pattern

To solve this, we'll create a dedicated **validation layer**. The `validator/` package is responsible for one thing and one thing only: validating raw, untrusted input from the outside world. It acts as a gatekeeper, ensuring that no invalid data ever reaches your business logic.

This validator is a pure function of its inputs, with no side effects and no knowledge of business logic or databases.

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

The `URLService` is the consumer of this validator. Before performing any action, it first ensures the input is sane. This is the **Gatekeeper Pattern**.

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

By centralizing validation, we ensure that our rules are applied consistently across all use cases. If we need to update our URL validation logic (e.g., to block more domains), we only need to change it in one place. This makes the system more secure and easier to maintain. The service layer can trust that any data it receives has already been vetted, allowing it to focus purely on business logic.

### Types of Validation Implemented

Our `URLValidator` is responsible for several distinct checks, each serving a critical purpose:

1.  **Syntactical Validation:** We use `url.ParseRequestURI` to ensure the URL is structurally valid according to RFC 3986. This is the first line of defense against malformed input.

2.  **Protocol Validation:** We explicitly check that the URL scheme is either `http` or `https`. This prevents abuse with other schemes like `ftp://`, `file://`, or potentially malicious custom schemes.

3.  **Semantic Validation (Domain Blocking):** This is a crucial security feature. Our validator checks the URL's host against a configurable list of blocked domains. This allows us to prevent our service from being used to shorten links to known malware sites, phishing pages, or internal resources. This list is managed in our `constants` package, making it easy to update without changing the validation logic itself.

4.  **Length Constraints:** We enforce a maximum length for both the original URL and any custom short codes. This is a simple but effective way to prevent denial-of-service attacks that attempt to exhaust our database storage with absurdly long strings.

5.  **Character Set Validation:** For custom short codes, we enforce an alphanumeric character set. This prevents users from injecting special characters, control characters, or potentially harmful script tags into our URLs.

By performing these checks in a dedicated, centralized validator, we create a single source of truth for our application's input rules. This makes the system more secure, more predictable, and easier to audit.

---

## Talking to the Outside World: A Resilient PocketBase Repository

With our internal structure sorted, it's time to connect to the outside world. The repository layer is the bridge between our application's domain and the persistence layer. It's also a place where it's easy to create leaky abstractions.

### The Anti-Pattern: The Leaky Repository

A common mistake is to write a repository that doesn't fully abstract away the details of the external service it's communicating with. The logic of HTTP requests, status codes, and external data formats can bleed into the service layer.

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

This approach breaks our clean architecture:

1.  **Leaky Abstraction:** The service layer is now coupled to the implementation details of PocketBase. It has to know what the request body should look like and how to interpret HTTP status codes. This violates the Dependency Rule.
2.  **Reduced Testability:** Testing the service now requires mocking the `http.Response`, which is cumbersome. The service should only need to know about our domain models and errors.
3.  **Difficult to Swap:** If we wanted to replace PocketBase with a direct SQL database, we would have to rewrite significant portions of our service layer, not just the repository.

### The Solution: The Adapter Pattern

Our repository should be a true **Adapter**. Its job is to adapt the interface of our application (the `URLRepository` interface, which speaks in our domain models) to the interface of the external service (the PocketBase REST API, which speaks in HTTP and JSON). The service layer should be completely shielded from these external details.

This choice informs a critical architectural decision: we will treat PocketBase not as an embedded database, but as a remote, external service. Our application will communicate with it exclusively over HTTP. This approach has profound benefits for maintainability and scalability:

1.  **True Decoupling:** Our Go service is now completely decoupled from its storage backend. The repository layer's job is to be an expert HTTP client for the PocketBase API. We could swap PocketBase for a different database service (like Supabase or a custom REST API) simply by writing a new repository that conforms to the `URLRepository` interface. The core business logic would remain untouched.

2.  **Leveraging External Features:** PocketBase offers more than just storage. It has its own validation rules, authentication, and a real-time event system. By communicating via its API, we can leverage these features without having to implement them ourselves in our Go service.

3.  **Forced Cleanliness:** The network boundary is a powerful forcing function. It prevents us from writing leaky abstractions where database-specific logic (like SQL queries or transaction management) bleeds into our business layer. Our repository must translate our internal domain models into HTTP requests and translate HTTP responses back into our domain models and errors.

Let's examine the concrete implementation of our `urlRepositoryImpl` to see this in practice.

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

This repository is a perfect example of an **Adapter**. It adapts our application's internal interfaces to the external interface of the PocketBase API. It is the sole gatekeeper of this external communication, creating a clean and testable boundary between our service and the outside world.

---

## The Lifeline of a Request: Mastering `context.Context`

If there's one pattern that separates a professional Go service from an amateur one, it's the disciplined use of `context.Context`. Neglecting it is one of the most common sources of bugs, performance degradation, and cascading failures in distributed systems.

Imagine a user sends a request to your service, which then makes a call to a downstream API (like PocketBase). What happens if the user closes their browser? Or if the downstream API hangs and never responds? Without `context`, your server is left in the dark. It will continue to process the request, hold the connection open, and consume memory and CPU cycles, all for a result that no one is waiting for. Now, multiply this by thousands of requests per second. This is how you build a service that will reliably fall over under pressure.

`context` is Go's elegant solution to this problem. It provides a request-scoped "lifeline" that carries cancellation signals, deadlines, and other values across API boundaries. It must be the first argument to any function that is part of a request's call chain, especially those involving I/O.

### Our Context Propagation Strategy

We implemented a simple but powerful context strategy:

1.  **Origination in the Handler:** The lifecycle of our context begins in the HTTP handler. The Gin framework automatically provides a `context.Context` for each incoming request, which we can access via `c.Request.Context()`. This context is automatically cancelled if the client disconnects.

2.  **Propagation Through the Service:** Every method in our `URLService` interface accepts `ctx` as its first argument. It does not inspect or modify the context; it simply acts as a carrier, passing it down to the repository layer.

3.  **Termination in the Repository:** The repository is where the context is finally consumed. It uses the context to create deadline-aware and cancellable outbound HTTP requests.

Let's look at the critical piece of code in our repository:

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

This implementation gives us two crucial guarantees:

-   **Fail-Fast on Timeouts:** We will never wait more than 30 seconds for PocketBase to respond. This prevents slow downstream services from causing a cascading failure in our application.
-   **Work Cancellation:** If the initial HTTP request is cancelled by the user, that cancellation signal will propagate down and cause our `http.Client` to immediately abort the outbound request to PocketBase. We stop wasting resources the instant the work is no longer needed.

This disciplined use of `context` is the bedrock of a resilient Go service.

---

## From Prototype to Better Code: A Blueprint for Your Next Service

We started with a common scenario: a simple script that solved a problem but was riddled with technical debt. Through a series of deliberate engineering decisions, we transformed it into a robust service that is a pleasure to work on. This wasn't about premature optimization or adding layers for the sake of it; it was about making pragmatic choices that lead to a more maintainable, testable, and resilient system.

Let's recap the actionable principles you can apply to your next Go project:

1.  **Isolate Your Business Logic:** Your core logic should have no knowledge of your web framework, database, or any other external concern. Encapsulate it in a `service` layer that depends only on interfaces.

2.  **Define Contracts with Interfaces:** Use interfaces to define the boundaries between your layers (`URLService`, `URLRepository`). This is the key to testability and modularity.

3.  **Separate Public and Private Models:** Use DTOs to define your public API contract. This decouples your API from your internal data structures, preventing breaking changes and information leakage.

4.  **Treat Errors as Structured Data:** Don't just return `error` strings. Return custom error types that contain rich, structured context. This allows for consistent error handling and powerful, structured logging.

5.  **Validate at the Edge:** Create a dedicated validation layer to act as a gatekeeper for all incoming requests. Your business logic should only ever operate on data that has been proven to be safe and valid.

6.  **Propagate Context Everywhere:** Make `context.Context` the first argument to every function in your request path. This is your lifeline for handling timeouts, cancellations, and building resilient systems.

By embracing these principles, you move from simply writing code to engineering a system. You build a well-organized workshop where any engineer can confidently and safely contribute, rather than a chaotic garage where only the original creator knows where the tools are. The initial investment in this structure pays dividends in reduced bugs, faster feature development, and a more maintainable codebase.

---

## Conclusion

We began with a familiar scenario: a monolithic handler that worked but couldn't scale. Through our systematic application of clean architecture principles, we've transformed it into a resilient, maintainable service. This transformation wasn't about adding complexity—it was about organizing complexity in a way that actually makes our lives easier.

Our URL shortener now demonstrates six critical patterns that you can apply to any Go service: domain isolation through service layers, interface-based contracts for testability, explicit API boundaries via DTOs, structured error handling for observability, centralized input validation for security, and context-aware operations for resilience.

Here's my advice: apply these patterns incrementally to your existing services. Start with error handling and input validation—they'll give you immediate value with minimal refactoring. Then introduce service interfaces and DTOs to create those testable boundaries we've been talking about. Save the repository pattern for last, as it requires the most architectural change but provides the greatest long-term flexibility.

Remember, the goal isn't perfection from day one. It's building systems that can evolve safely, scale predictably, and welcome new contributors without requiring archaeological expeditions through legacy code. You've got this!

---

## Project Resources

- **GitHub Repository:** [github.com/rowjay007/url-shortening-service](https://github.com/rowjay007/url-shortening-service)
- **Original Project Brief:** [roadmap.sh/projects/url-shortening-service](https://roadmap.sh/projects/url-shortening-service)
- **Author:** [@rowjay](https://github.com/rowjay007)
