# Plan 2: User Authentication

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Prerequisite:** Plan 01-setup.md complete.

**Goal:** Implement user registration and login with bcrypt password hashing and JWT token generation. Includes auth middleware for protecting routes.

**Architecture:** `UserRepository` (SQL) → `AuthService` (business logic + JWT) → `AuthHandler` (HTTP) + `middleware.Auth` (JWT guard).

**Tech Stack:** `golang-jwt/jwt`, `bcrypt`, `chi`.

---

## Files Created

```
internal/repository/user.go
internal/service/auth.go
internal/service/auth_test.go
internal/middleware/auth.go
internal/handler/auth.go
```

---

## Task 1: User Repository

- [ ] **Step 1: Create `internal/repository/user.go`**

```go
package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/aman-chandra314/cinema-booking-system/internal/domain"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, u domain.User) (domain.User, error) {
	query := `
		INSERT INTO users (email, password_hash, name)
		VALUES ($1, $2, $3)
		RETURNING id, email, password_hash, name, created_at`
	row := r.db.QueryRowContext(ctx, query, u.Email, u.PasswordHash, u.Name)
	if err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.CreatedAt); err != nil {
		return domain.User{}, fmt.Errorf("create user: %w", err)
	}
	return u, nil
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (domain.User, error) {
	query := `SELECT id, email, password_hash, name, created_at FROM users WHERE email = $1`
	var u domain.User
	if err := r.db.QueryRowContext(ctx, query, email).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.CreatedAt); err != nil {
		return domain.User{}, fmt.Errorf("find user by email: %w", err)
	}
	return u, nil
}

func (r *UserRepository) FindByID(ctx context.Context, id string) (domain.User, error) {
	query := `SELECT id, email, password_hash, name, created_at FROM users WHERE id = $1`
	var u domain.User
	if err := r.db.QueryRowContext(ctx, query, id).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.CreatedAt); err != nil {
		return domain.User{}, fmt.Errorf("find user by id: %w", err)
	}
	return u, nil
}
```

- [ ] **Step 2: Verify compilation**

```bash
go build ./...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/repository/user.go
git commit -m "feat: add user repository"
```

---

## Task 2: Auth Service

- [ ] **Step 1: Write failing tests — create `internal/service/auth_test.go`**

```go
package service_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/aman-chandra314/cinema-booking-system/internal/domain"
	"github.com/aman-chandra314/cinema-booking-system/internal/service"
)

type mockUserRepo struct {
	createFn      func(ctx context.Context, u domain.User) (domain.User, error)
	findByEmailFn func(ctx context.Context, email string) (domain.User, error)
	findByIDFn    func(ctx context.Context, id string) (domain.User, error)
}

func (m *mockUserRepo) Create(ctx context.Context, u domain.User) (domain.User, error) {
	return m.createFn(ctx, u)
}
func (m *mockUserRepo) FindByEmail(ctx context.Context, email string) (domain.User, error) {
	return m.findByEmailFn(ctx, email)
}
func (m *mockUserRepo) FindByID(ctx context.Context, id string) (domain.User, error) {
	return m.findByIDFn(ctx, id)
}

func TestRegister_Success(t *testing.T) {
	repo := &mockUserRepo{
		findByEmailFn: func(ctx context.Context, email string) (domain.User, error) {
			return domain.User{}, sql.ErrNoRows
		},
		createFn: func(ctx context.Context, u domain.User) (domain.User, error) {
			u.ID = "user-1"
			return u, nil
		},
	}
	svc := service.NewAuthService(repo, "test-secret")
	token, err := svc.Register(context.Background(), "Alice", "alice@example.com", "password123")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if token == "" {
		t.Fatal("expected a JWT token, got empty string")
	}
}

func TestRegister_DuplicateEmail(t *testing.T) {
	repo := &mockUserRepo{
		findByEmailFn: func(ctx context.Context, email string) (domain.User, error) {
			return domain.User{ID: "existing"}, nil
		},
	}
	svc := service.NewAuthService(repo, "test-secret")
	_, err := svc.Register(context.Background(), "Alice", "alice@example.com", "password123")
	if err == nil {
		t.Fatal("expected error for duplicate email, got nil")
	}
}

func TestLogin_Success(t *testing.T) {
	svc := service.NewAuthService(nil, "test-secret")
	hash, _ := svc.HashPassword("password123")
	repo := &mockUserRepo{
		findByEmailFn: func(ctx context.Context, email string) (domain.User, error) {
			return domain.User{ID: "user-1", Email: email, PasswordHash: hash}, nil
		},
	}
	svc = service.NewAuthService(repo, "test-secret")
	token, err := svc.Login(context.Background(), "alice@example.com", "password123")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if token == "" {
		t.Fatal("expected a JWT token, got empty string")
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	svc := service.NewAuthService(nil, "test-secret")
	hash, _ := svc.HashPassword("correct-password")
	repo := &mockUserRepo{
		findByEmailFn: func(ctx context.Context, email string) (domain.User, error) {
			return domain.User{ID: "user-1", Email: email, PasswordHash: hash}, nil
		},
	}
	svc = service.NewAuthService(repo, "test-secret")
	_, err := svc.Login(context.Background(), "alice@example.com", "wrong-password")
	if err == nil {
		t.Fatal("expected error for wrong password, got nil")
	}
}
```

- [ ] **Step 2: Run tests — verify they fail**

```bash
go test ./internal/service/... -v
```

Expected: compile error — `service.NewAuthService` not defined yet.

- [ ] **Step 3: Implement `internal/service/auth.go`**

```go
package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/aman-chandra314/cinema-booking-system/internal/domain"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type UserRepo interface {
	Create(ctx context.Context, u domain.User) (domain.User, error)
	FindByEmail(ctx context.Context, email string) (domain.User, error)
	FindByID(ctx context.Context, id string) (domain.User, error)
}

type AuthService struct {
	users     UserRepo
	jwtSecret string
}

func NewAuthService(users UserRepo, jwtSecret string) *AuthService {
	return &AuthService{users: users, jwtSecret: jwtSecret}
}

func (s *AuthService) HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(hash), nil
}

func (s *AuthService) Register(ctx context.Context, name, email, password string) (string, error) {
	existing, err := s.users.FindByEmail(ctx, email)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("check existing user: %w", err)
	}
	if existing.ID != "" {
		return "", errors.New("email already registered")
	}

	hash, err := s.HashPassword(password)
	if err != nil {
		return "", err
	}

	user, err := s.users.Create(ctx, domain.User{Name: name, Email: email, PasswordHash: hash})
	if err != nil {
		return "", fmt.Errorf("create user: %w", err)
	}
	return s.generateToken(user.ID)
}

func (s *AuthService) Login(ctx context.Context, email, password string) (string, error) {
	user, err := s.users.FindByEmail(ctx, email)
	if err != nil {
		return "", errors.New("invalid email or password")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", errors.New("invalid email or password")
	}
	return s.generateToken(user.ID)
}

func (s *AuthService) ValidateToken(tokenStr string) (string, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(s.jwtSecret), nil
	})
	if err != nil || !token.Valid {
		return "", errors.New("invalid token")
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", errors.New("invalid token claims")
	}
	userID, ok := claims["sub"].(string)
	if !ok || userID == "" {
		return "", errors.New("invalid token subject")
	}
	return userID, nil
}

func (s *AuthService) generateToken(userID string) (string, error) {
	claims := jwt.MapClaims{
		"sub": userID,
		"exp": time.Now().Add(24 * time.Hour).Unix(),
		"iat": time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return signed, nil
}
```

- [ ] **Step 4: Run tests — verify they pass**

```bash
go test ./internal/service/... -v
```

Expected: 4 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/service/auth.go internal/service/auth_test.go
git commit -m "feat: add auth service with JWT and bcrypt"
```

---

## Task 3: Auth Middleware

- [ ] **Step 1: Create `internal/middleware/auth.go`**

```go
package middleware

import (
	"context"
	"net/http"
	"strings"
)

type contextKey string

const UserIDKey contextKey = "userID"

type TokenValidator interface {
	ValidateToken(token string) (string, error)
}

func Auth(validator TokenValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				http.Error(w, `{"error":"missing or invalid authorization header"}`, http.StatusUnauthorized)
				return
			}
			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
			userID, err := validator.ValidateToken(tokenStr)
			if err != nil {
				http.Error(w, `{"error":"invalid or expired token"}`, http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), UserIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetUserID(ctx context.Context) string {
	id, _ := ctx.Value(UserIDKey).(string)
	return id
}
```

- [ ] **Step 2: Verify compilation**

```bash
go build ./...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/middleware/
git commit -m "feat: add JWT auth middleware"
```

---

## Task 4: Auth Handler + Wire Routes

- [ ] **Step 1: Create `internal/handler/auth.go`**

```go
package handler

import (
	"context"
	"encoding/json"
	"net/http"
)

type AuthServiceIface interface {
	Register(ctx context.Context, name, email, password string) (string, error)
	Login(ctx context.Context, email, password string) (string, error)
}

type AuthHandler struct {
	auth AuthServiceIface
}

func NewAuthHandler(auth AuthServiceIface) *AuthHandler {
	return &AuthHandler{auth: auth}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" || req.Email == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "name, email and password are required")
		return
	}
	token, err := h.auth.Register(r.Context(), req.Name, req.Email, req.Password)
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"token": token})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Email == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "email and password are required")
		return
	}
	token, err := h.auth.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"token": token})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
```

- [ ] **Step 2: Update `cmd/api/main.go` to wire auth routes**

```go
package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"github.com/aman-chandra314/cinema-booking-system/configs"
	"github.com/aman-chandra314/cinema-booking-system/internal/db"
	"github.com/aman-chandra314/cinema-booking-system/internal/handler"
	"github.com/aman-chandra314/cinema-booking-system/internal/repository"
	"github.com/aman-chandra314/cinema-booking-system/internal/service"
)

func main() {
	cfg := configs.Load()

	database, err := db.Connect(cfg.DBURL)
	if err != nil {
		log.Fatalf("connect to db: %v", err)
	}
	defer database.Close()

	userRepo := repository.NewUserRepository(database)
	authSvc := service.NewAuthService(userRepo, cfg.JWTSecret)
	authHandler := handler.NewAuthHandler(authSvc)

	r := chi.NewRouter()
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/auth/register", authHandler.Register)
		r.Post("/auth/login", authHandler.Login)
	})

	log.Printf("server starting on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
```

- [ ] **Step 3: Smoke test**

```bash
go run ./cmd/api &

curl -s -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"name":"Alice","email":"alice@example.com","password":"password123"}' | jq .
```

Expected: `{"token": "<jwt>"}` with HTTP 201.

```bash
curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"alice@example.com","password":"password123"}' | jq .
```

Expected: `{"token": "<jwt>"}` with HTTP 200.

- [ ] **Step 4: Commit**

```bash
git add internal/handler/auth.go cmd/api/main.go
git commit -m "feat: auth handler and routes wired"
```
