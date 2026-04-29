# Plan 4: Showtimes

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Prerequisite:** Plan 03-movies-cinemas.md complete.

**Goal:** Implement showtime listing (with movie/date filters), showtime detail, and available seat lookup. All routes are public.

**Architecture:** `ShowtimeRepository` (SQL) → `ShowtimeService` → `ShowtimeHandler` (HTTP).

**Tech Stack:** Go, `chi`, `database/sql`.

---

## Files Created

```
internal/repository/showtime.go
internal/service/showtime.go
internal/handler/showtime.go
```

`cmd/api/main.go` modified to add showtime routes.

---

## Task 1: Showtime Repository

- [ ] **Step 1: Create `internal/repository/showtime.go`**

```go
package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/aman-chandra314/cinema-booking-system/internal/domain"
)

type ShowtimeRepository struct {
	db *sql.DB
}

func NewShowtimeRepository(db *sql.DB) *ShowtimeRepository {
	return &ShowtimeRepository{db: db}
}

func (r *ShowtimeRepository) List(ctx context.Context, movieID, date string) ([]domain.Showtime, error) {
	query := `SELECT id, movie_id, hall_id, start_time, end_time, price FROM showtimes WHERE 1=1`
	args := []interface{}{}
	i := 1
	if movieID != "" {
		query += fmt.Sprintf(" AND movie_id = $%d", i)
		args = append(args, movieID)
		i++
	}
	if date != "" {
		query += fmt.Sprintf(" AND DATE(start_time) = $%d", i)
		args = append(args, date)
	}
	query += " ORDER BY start_time"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list showtimes: %w", err)
	}
	defer rows.Close()

	var showtimes []domain.Showtime
	for rows.Next() {
		var s domain.Showtime
		if err := rows.Scan(&s.ID, &s.MovieID, &s.HallID, &s.StartTime, &s.EndTime, &s.Price); err != nil {
			return nil, fmt.Errorf("scan showtime: %w", err)
		}
		showtimes = append(showtimes, s)
	}
	return showtimes, rows.Err()
}

func (r *ShowtimeRepository) FindByID(ctx context.Context, id string) (domain.Showtime, error) {
	query := `SELECT id, movie_id, hall_id, start_time, end_time, price FROM showtimes WHERE id = $1`
	var s domain.Showtime
	if err := r.db.QueryRowContext(ctx, query, id).Scan(&s.ID, &s.MovieID, &s.HallID, &s.StartTime, &s.EndTime, &s.Price); err != nil {
		return domain.Showtime{}, fmt.Errorf("find showtime by id: %w", err)
	}
	return s, nil
}

func (r *ShowtimeRepository) GetAvailableSeats(ctx context.Context, showtimeID string) ([]domain.Seat, error) {
	query := `
		SELECT s.id, s.hall_id, s.row, s.number
		FROM seats s
		JOIN showtimes st ON st.hall_id = s.hall_id
		WHERE st.id = $1
		  AND s.id NOT IN (
		      SELECT bs.seat_id FROM booking_seats bs
		      JOIN bookings b ON b.id = bs.booking_id
		      WHERE b.showtime_id = $1 AND b.status = 'confirmed'
		  )
		ORDER BY s.row, s.number`
	rows, err := r.db.QueryContext(ctx, query, showtimeID)
	if err != nil {
		return nil, fmt.Errorf("get available seats: %w", err)
	}
	defer rows.Close()

	var seats []domain.Seat
	for rows.Next() {
		var s domain.Seat
		if err := rows.Scan(&s.ID, &s.HallID, &s.Row, &s.Number); err != nil {
			return nil, fmt.Errorf("scan seat: %w", err)
		}
		seats = append(seats, s)
	}
	return seats, rows.Err()
}

func (r *ShowtimeRepository) GetBookedSeatIDs(ctx context.Context, showtimeID string) ([]string, error) {
	query := `
		SELECT bs.seat_id FROM booking_seats bs
		JOIN bookings b ON b.id = bs.booking_id
		WHERE b.showtime_id = $1 AND b.status = 'confirmed'`
	rows, err := r.db.QueryContext(ctx, query, showtimeID)
	if err != nil {
		return nil, fmt.Errorf("get booked seat ids: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan seat id: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
```

- [ ] **Step 2: Verify compilation**

```bash
go build ./...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/repository/showtime.go
git commit -m "feat: add showtime repository"
```

---

## Task 2: Showtime Service + Handler

- [ ] **Step 1: Create `internal/service/showtime.go`**

```go
package service

import (
	"context"
	"fmt"

	"github.com/aman-chandra314/cinema-booking-system/internal/domain"
)

type ShowtimeRepo interface {
	List(ctx context.Context, movieID, date string) ([]domain.Showtime, error)
	FindByID(ctx context.Context, id string) (domain.Showtime, error)
	GetAvailableSeats(ctx context.Context, showtimeID string) ([]domain.Seat, error)
	GetBookedSeatIDs(ctx context.Context, showtimeID string) ([]string, error)
}

type ShowtimeService struct {
	showtimes ShowtimeRepo
}

func NewShowtimeService(showtimes ShowtimeRepo) *ShowtimeService {
	return &ShowtimeService{showtimes: showtimes}
}

func (s *ShowtimeService) List(ctx context.Context, movieID, date string) ([]domain.Showtime, error) {
	return s.showtimes.List(ctx, movieID, date)
}

func (s *ShowtimeService) Get(ctx context.Context, id string) (domain.Showtime, error) {
	st, err := s.showtimes.FindByID(ctx, id)
	if err != nil {
		return domain.Showtime{}, fmt.Errorf("showtime not found: %w", err)
	}
	return st, nil
}

func (s *ShowtimeService) GetAvailableSeats(ctx context.Context, showtimeID string) ([]domain.Seat, error) {
	return s.showtimes.GetAvailableSeats(ctx, showtimeID)
}
```

- [ ] **Step 2: Create `internal/handler/showtime.go`**

```go
package handler

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/aman-chandra314/cinema-booking-system/internal/domain"
)

type ShowtimeServiceIface interface {
	List(ctx context.Context, movieID, date string) ([]domain.Showtime, error)
	Get(ctx context.Context, id string) (domain.Showtime, error)
	GetAvailableSeats(ctx context.Context, showtimeID string) ([]domain.Seat, error)
}

type ShowtimeHandler struct {
	showtimes ShowtimeServiceIface
}

func NewShowtimeHandler(showtimes ShowtimeServiceIface) *ShowtimeHandler {
	return &ShowtimeHandler{showtimes: showtimes}
}

func (h *ShowtimeHandler) List(w http.ResponseWriter, r *http.Request) {
	movieID := r.URL.Query().Get("movie_id")
	date := r.URL.Query().Get("date")
	showtimes, err := h.showtimes.List(r.Context(), movieID, date)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if showtimes == nil {
		showtimes = []domain.Showtime{}
	}
	writeJSON(w, http.StatusOK, showtimes)
}

func (h *ShowtimeHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	showtime, err := h.showtimes.Get(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "showtime not found")
		return
	}
	writeJSON(w, http.StatusOK, showtime)
}

func (h *ShowtimeHandler) GetAvailableSeats(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	seats, err := h.showtimes.GetAvailableSeats(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if seats == nil {
		seats = []domain.Seat{}
	}
	writeJSON(w, http.StatusOK, seats)
}
```

- [ ] **Step 3: Verify compilation**

```bash
go build ./...
```

Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add internal/service/showtime.go internal/handler/showtime.go
git commit -m "feat: add showtime service and handler"
```

---

## Task 3: Wire Showtime Routes + Smoke Test

- [ ] **Step 1: Update `cmd/api/main.go` — add showtime routes**

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
	showtimeRepo := repository.NewShowtimeRepository(database)

	authSvc := service.NewAuthService(userRepo, cfg.JWTSecret)
	movieSvc := service.NewMovieService(movieRepo)
	cinemaSvc := service.NewCinemaService(cinemaRepo)
	showtimeSvc := service.NewShowtimeService(showtimeRepo)

	authHandler := handler.NewAuthHandler(authSvc)
	movieHandler := handler.NewMovieHandler(movieSvc)
	cinemaHandler := handler.NewCinemaHandler(cinemaSvc)
	showtimeHandler := handler.NewShowtimeHandler(showtimeSvc)

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

		r.Get("/showtimes", showtimeHandler.List)
		r.Get("/showtimes/{id}", showtimeHandler.Get)
		r.Get("/showtimes/{id}/seats", showtimeHandler.GetAvailableSeats)
	})

	log.Printf("server starting on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
```

- [ ] **Step 2: Seed data and smoke test**

```bash
docker exec -i $(docker ps -qf "name=postgres") psql -U postgres -d cinema_booking <<'SQL'
INSERT INTO cinemas (name, location) VALUES ('CineMax', 'Downtown');
INSERT INTO halls (cinema_id, name, total_seats)
  SELECT id, 'Hall A', 50 FROM cinemas WHERE name = 'CineMax';
INSERT INTO seats (hall_id, row, number)
  SELECT id, 'A', generate_series(1,10) FROM halls WHERE name = 'Hall A';
INSERT INTO showtimes (movie_id, hall_id, start_time, end_time, price)
  SELECT m.id, h.id, NOW() + interval '2 hours', NOW() + interval '4 hours', 12.50
  FROM movies m, halls h WHERE m.title = 'Inception' AND h.name = 'Hall A';
SQL

go run ./cmd/api &

curl -s http://localhost:8080/api/v1/showtimes | jq .
```

Expected: array with 1 showtime.

```bash
SHOWTIME_ID=$(curl -s http://localhost:8080/api/v1/showtimes | jq -r '.[0].id')
curl -s http://localhost:8080/api/v1/showtimes/$SHOWTIME_ID/seats | jq .
```

Expected: 10 seats listed (A1–A10).

- [ ] **Step 3: Commit**

```bash
git add cmd/api/main.go
git commit -m "feat: wire showtime routes"
```
