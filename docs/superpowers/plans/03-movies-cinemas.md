# Plan 3: Movies & Cinemas

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Prerequisite:** Plan 02-auth.md complete.

**Goal:** Implement list and get endpoints for movies and cinemas (including halls). All routes are public (no auth required).

**Architecture:** `MovieRepository` + `CinemaRepository` (SQL) → `MovieService` + `CinemaService` → `MovieHandler` + `CinemaHandler` (HTTP).

**Tech Stack:** Go, `chi`, `database/sql`.

---

## Files Created

```
internal/repository/movie.go
internal/repository/cinema.go
internal/service/movie.go
internal/service/cinema.go
internal/handler/movie.go
internal/handler/cinema.go
```

`cmd/api/main.go` modified to add new routes.

---

## Task 1: Movie Repository + Service + Handler

- [ ] **Step 1: Create `internal/repository/movie.go`**

```go
package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/aman-chandra314/cinema-booking-system/internal/domain"
)

type MovieRepository struct {
	db *sql.DB
}

func NewMovieRepository(db *sql.DB) *MovieRepository {
	return &MovieRepository{db: db}
}

func (r *MovieRepository) List(ctx context.Context) ([]domain.Movie, error) {
	query := `SELECT id, title, description, duration_minutes, genre, release_date FROM movies ORDER BY title`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list movies: %w", err)
	}
	defer rows.Close()

	var movies []domain.Movie
	for rows.Next() {
		var m domain.Movie
		if err := rows.Scan(&m.ID, &m.Title, &m.Description, &m.DurationMinutes, &m.Genre, &m.ReleaseDate); err != nil {
			return nil, fmt.Errorf("scan movie: %w", err)
		}
		movies = append(movies, m)
	}
	return movies, rows.Err()
}

func (r *MovieRepository) FindByID(ctx context.Context, id string) (domain.Movie, error) {
	query := `SELECT id, title, description, duration_minutes, genre, release_date FROM movies WHERE id = $1`
	var m domain.Movie
	if err := r.db.QueryRowContext(ctx, query, id).Scan(&m.ID, &m.Title, &m.Description, &m.DurationMinutes, &m.Genre, &m.ReleaseDate); err != nil {
		return domain.Movie{}, fmt.Errorf("find movie by id: %w", err)
	}
	return m, nil
}
```

- [ ] **Step 2: Create `internal/service/movie.go`**

```go
package service

import (
	"context"
	"fmt"

	"github.com/aman-chandra314/cinema-booking-system/internal/domain"
)

type MovieRepo interface {
	List(ctx context.Context) ([]domain.Movie, error)
	FindByID(ctx context.Context, id string) (domain.Movie, error)
}

type MovieService struct {
	movies MovieRepo
}

func NewMovieService(movies MovieRepo) *MovieService {
	return &MovieService{movies: movies}
}

func (s *MovieService) List(ctx context.Context) ([]domain.Movie, error) {
	return s.movies.List(ctx)
}

func (s *MovieService) Get(ctx context.Context, id string) (domain.Movie, error) {
	m, err := s.movies.FindByID(ctx, id)
	if err != nil {
		return domain.Movie{}, fmt.Errorf("movie not found: %w", err)
	}
	return m, nil
}
```

- [ ] **Step 3: Create `internal/handler/movie.go`**

```go
package handler

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/aman-chandra314/cinema-booking-system/internal/domain"
)

type MovieServiceIface interface {
	List(ctx context.Context) ([]domain.Movie, error)
	Get(ctx context.Context, id string) (domain.Movie, error)
}

type MovieHandler struct {
	movies MovieServiceIface
}

func NewMovieHandler(movies MovieServiceIface) *MovieHandler {
	return &MovieHandler{movies: movies}
}

func (h *MovieHandler) List(w http.ResponseWriter, r *http.Request) {
	movies, err := h.movies.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if movies == nil {
		movies = []domain.Movie{}
	}
	writeJSON(w, http.StatusOK, movies)
}

func (h *MovieHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	movie, err := h.movies.Get(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "movie not found")
		return
	}
	writeJSON(w, http.StatusOK, movie)
}
```

- [ ] **Step 4: Verify compilation**

```bash
go build ./...
```

Expected: no errors.

- [ ] **Step 5: Commit**

```bash
git add internal/repository/movie.go internal/service/movie.go internal/handler/movie.go
git commit -m "feat: add movie repository, service, and handler"
```

---

## Task 2: Cinema Repository + Service + Handler

- [ ] **Step 1: Create `internal/repository/cinema.go`**

```go
package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/aman-chandra314/cinema-booking-system/internal/domain"
)

type CinemaRepository struct {
	db *sql.DB
}

func NewCinemaRepository(db *sql.DB) *CinemaRepository {
	return &CinemaRepository{db: db}
}

func (r *CinemaRepository) List(ctx context.Context) ([]domain.Cinema, error) {
	query := `SELECT id, name, location, created_at FROM cinemas ORDER BY name`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list cinemas: %w", err)
	}
	defer rows.Close()

	var cinemas []domain.Cinema
	for rows.Next() {
		var c domain.Cinema
		if err := rows.Scan(&c.ID, &c.Name, &c.Location, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan cinema: %w", err)
		}
		cinemas = append(cinemas, c)
	}
	return cinemas, rows.Err()
}

func (r *CinemaRepository) FindByID(ctx context.Context, id string) (domain.Cinema, error) {
	query := `SELECT id, name, location, created_at FROM cinemas WHERE id = $1`
	var c domain.Cinema
	if err := r.db.QueryRowContext(ctx, query, id).Scan(&c.ID, &c.Name, &c.Location, &c.CreatedAt); err != nil {
		return domain.Cinema{}, fmt.Errorf("find cinema by id: %w", err)
	}

	hallQuery := `SELECT id, cinema_id, name, total_seats FROM halls WHERE cinema_id = $1 ORDER BY name`
	rows, err := r.db.QueryContext(ctx, hallQuery, id)
	if err != nil {
		return domain.Cinema{}, fmt.Errorf("list halls: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var h domain.Hall
		if err := rows.Scan(&h.ID, &h.CinemaID, &h.Name, &h.TotalSeats); err != nil {
			return domain.Cinema{}, fmt.Errorf("scan hall: %w", err)
		}
		c.Halls = append(c.Halls, h)
	}
	return c, rows.Err()
}
```

- [ ] **Step 2: Create `internal/service/cinema.go`**

```go
package service

import (
	"context"
	"fmt"

	"github.com/aman-chandra314/cinema-booking-system/internal/domain"
)

type CinemaRepo interface {
	List(ctx context.Context) ([]domain.Cinema, error)
	FindByID(ctx context.Context, id string) (domain.Cinema, error)
}

type CinemaService struct {
	cinemas CinemaRepo
}

func NewCinemaService(cinemas CinemaRepo) *CinemaService {
	return &CinemaService{cinemas: cinemas}
}

func (s *CinemaService) List(ctx context.Context) ([]domain.Cinema, error) {
	return s.cinemas.List(ctx)
}

func (s *CinemaService) Get(ctx context.Context, id string) (domain.Cinema, error) {
	c, err := s.cinemas.FindByID(ctx, id)
	if err != nil {
		return domain.Cinema{}, fmt.Errorf("cinema not found: %w", err)
	}
	return c, nil
}
```

- [ ] **Step 3: Create `internal/handler/cinema.go`**

```go
package handler

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/aman-chandra314/cinema-booking-system/internal/domain"
)

type CinemaServiceIface interface {
	List(ctx context.Context) ([]domain.Cinema, error)
	Get(ctx context.Context, id string) (domain.Cinema, error)
}

type CinemaHandler struct {
	cinemas CinemaServiceIface
}

func NewCinemaHandler(cinemas CinemaServiceIface) *CinemaHandler {
	return &CinemaHandler{cinemas: cinemas}
}

func (h *CinemaHandler) List(w http.ResponseWriter, r *http.Request) {
	cinemas, err := h.cinemas.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if cinemas == nil {
		cinemas = []domain.Cinema{}
	}
	writeJSON(w, http.StatusOK, cinemas)
}

func (h *CinemaHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	cinema, err := h.cinemas.Get(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "cinema not found")
		return
	}
	writeJSON(w, http.StatusOK, cinema)
}
```

- [ ] **Step 4: Verify compilation**

```bash
go build ./...
```

Expected: no errors.

- [ ] **Step 5: Commit**

```bash
git add internal/repository/cinema.go internal/service/cinema.go internal/handler/cinema.go
git commit -m "feat: add cinema repository, service, and handler"
```

---

## Task 3: Wire Movie + Cinema Routes

- [ ] **Step 1: Update `cmd/api/main.go` — add movie and cinema routes**

Add to the existing main.go (keep auth wiring, add the new repos/services/handlers):

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
	movieRepo := repository.NewMovieRepository(database)
	cinemaRepo := repository.NewCinemaRepository(database)

	authSvc := service.NewAuthService(userRepo, cfg.JWTSecret)
	movieSvc := service.NewMovieService(movieRepo)
	cinemaSvc := service.NewCinemaService(cinemaRepo)

	authHandler := handler.NewAuthHandler(authSvc)
	movieHandler := handler.NewMovieHandler(movieSvc)
	cinemaHandler := handler.NewCinemaHandler(cinemaSvc)

	r := chi.NewRouter()
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/auth/register", authHandler.Register)
		r.Post("/auth/login", authHandler.Login)

		r.Get("/movies", movieHandler.List)
		r.Get("/movies/{id}", movieHandler.Get)

		r.Get("/cinemas", cinemaHandler.List)
		r.Get("/cinemas/{id}", cinemaHandler.Get)
	})

	log.Printf("server starting on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
```

- [ ] **Step 2: Smoke test**

```bash
go run ./cmd/api &

# Seed a movie
docker exec -i $(docker ps -qf "name=postgres") psql -U postgres -d cinema_booking -c \
  "INSERT INTO movies (title, description, duration_minutes, genre) VALUES ('Inception', 'Mind-bending thriller', 148, 'Sci-Fi');"

curl -s http://localhost:8080/api/v1/movies | jq .
```

Expected: array with 1 movie.

```bash
curl -s http://localhost:8080/api/v1/cinemas | jq .
```

Expected: `[]` (no cinemas seeded yet).

- [ ] **Step 3: Commit**

```bash
git add cmd/api/main.go
git commit -m "feat: wire movie and cinema routes"
```
