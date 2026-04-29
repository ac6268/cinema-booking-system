# Plan 5: Bookings

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Prerequisite:** Plan 04-showtimes.md complete.

**Goal:** Implement create, list, get, and cancel booking endpoints. All routes are protected by JWT. Booking creation validates seat availability and runs inside a DB transaction to prevent double-booking.

**Architecture:** `BookingRepository` (SQL + transaction) → `BookingService` (business rules) → `BookingHandler` (HTTP). Auth middleware applied to all booking routes.

**Tech Stack:** Go, `chi`, `database/sql` transactions, JWT middleware.

---

## Files Created

```
internal/repository/booking.go
internal/service/booking.go
internal/service/booking_test.go
internal/handler/booking.go
```

`cmd/api/main.go` modified to add booking routes behind auth middleware.

---

## Task 1: Booking Repository

- [ ] **Step 1: Create `internal/repository/booking.go`**

```go
package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/aman-chandra314/cinema-booking-system/internal/domain"
)

type BookingRepository struct {
	db *sql.DB
}

func NewBookingRepository(db *sql.DB) *BookingRepository {
	return &BookingRepository{db: db}
}

func (r *BookingRepository) Create(ctx context.Context, b domain.Booking, seatIDs []string) (domain.Booking, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Booking{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	bookingQuery := `
		INSERT INTO bookings (user_id, showtime_id, total_price, status)
		VALUES ($1, $2, $3, 'confirmed')
		RETURNING id, user_id, showtime_id, total_price, status, created_at`
	row := tx.QueryRowContext(ctx, bookingQuery, b.UserID, b.ShowtimeID, b.TotalPrice)
	if err := row.Scan(&b.ID, &b.UserID, &b.ShowtimeID, &b.TotalPrice, &b.Status, &b.CreatedAt); err != nil {
		return domain.Booking{}, fmt.Errorf("insert booking: %w", err)
	}

	for _, seatID := range seatIDs {
		_, err := tx.ExecContext(ctx,
			`INSERT INTO booking_seats (booking_id, seat_id) VALUES ($1, $2)`,
			b.ID, seatID)
		if err != nil {
			return domain.Booking{}, fmt.Errorf("insert booking_seat: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return domain.Booking{}, fmt.Errorf("commit tx: %w", err)
	}
	return b, nil
}

func (r *BookingRepository) ListByUser(ctx context.Context, userID string) ([]domain.Booking, error) {
	query := `
		SELECT id, user_id, showtime_id, total_price, status, created_at
		FROM bookings WHERE user_id = $1 ORDER BY created_at DESC`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list bookings: %w", err)
	}
	defer rows.Close()

	var bookings []domain.Booking
	for rows.Next() {
		var b domain.Booking
		if err := rows.Scan(&b.ID, &b.UserID, &b.ShowtimeID, &b.TotalPrice, &b.Status, &b.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan booking: %w", err)
		}
		bookings = append(bookings, b)
	}
	return bookings, rows.Err()
}

func (r *BookingRepository) FindByID(ctx context.Context, id string) (domain.Booking, error) {
	query := `
		SELECT id, user_id, showtime_id, total_price, status, created_at
		FROM bookings WHERE id = $1`
	var b domain.Booking
	if err := r.db.QueryRowContext(ctx, query, id).Scan(&b.ID, &b.UserID, &b.ShowtimeID, &b.TotalPrice, &b.Status, &b.CreatedAt); err != nil {
		return domain.Booking{}, fmt.Errorf("find booking by id: %w", err)
	}

	seatQuery := `
		SELECT s.id, s.hall_id, s.row, s.number
		FROM seats s JOIN booking_seats bs ON bs.seat_id = s.id
		WHERE bs.booking_id = $1 ORDER BY s.row, s.number`
	rows, err := r.db.QueryContext(ctx, seatQuery, id)
	if err != nil {
		return domain.Booking{}, fmt.Errorf("get booking seats: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var s domain.Seat
		if err := rows.Scan(&s.ID, &s.HallID, &s.Row, &s.Number); err != nil {
			return domain.Booking{}, fmt.Errorf("scan seat: %w", err)
		}
		b.Seats = append(b.Seats, s)
	}
	return b, rows.Err()
}

func (r *BookingRepository) Cancel(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE bookings SET status = 'cancelled' WHERE id = $1 AND status = 'confirmed'`, id)
	if err != nil {
		return fmt.Errorf("cancel booking: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("booking not found or already cancelled")
	}
	return nil
}
```

- [ ] **Step 2: Verify compilation**

```bash
go build ./...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/repository/booking.go
git commit -m "feat: add booking repository with transaction support"
```

---

## Task 2: Booking Service (TDD)

- [ ] **Step 1: Write failing tests — create `internal/service/booking_test.go`**

```go
package service_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/aman-chandra314/cinema-booking-system/internal/domain"
	"github.com/aman-chandra314/cinema-booking-system/internal/service"
)

type mockBookingRepo struct {
	createFn     func(ctx context.Context, b domain.Booking, seatIDs []string) (domain.Booking, error)
	listByUserFn func(ctx context.Context, userID string) ([]domain.Booking, error)
	findByIDFn   func(ctx context.Context, id string) (domain.Booking, error)
	cancelFn     func(ctx context.Context, id string) error
}

func (m *mockBookingRepo) Create(ctx context.Context, b domain.Booking, seatIDs []string) (domain.Booking, error) {
	return m.createFn(ctx, b, seatIDs)
}
func (m *mockBookingRepo) ListByUser(ctx context.Context, userID string) ([]domain.Booking, error) {
	return m.listByUserFn(ctx, userID)
}
func (m *mockBookingRepo) FindByID(ctx context.Context, id string) (domain.Booking, error) {
	return m.findByIDFn(ctx, id)
}
func (m *mockBookingRepo) Cancel(ctx context.Context, id string) error {
	return m.cancelFn(ctx, id)
}

type mockShowtimeRepoForBooking struct {
	findByIDFn       func(ctx context.Context, id string) (domain.Showtime, error)
	getBookedSeatsFn func(ctx context.Context, showtimeID string) ([]string, error)
}

func (m *mockShowtimeRepoForBooking) List(ctx context.Context, movieID, date string) ([]domain.Showtime, error) {
	return nil, nil
}
func (m *mockShowtimeRepoForBooking) FindByID(ctx context.Context, id string) (domain.Showtime, error) {
	return m.findByIDFn(ctx, id)
}
func (m *mockShowtimeRepoForBooking) GetAvailableSeats(ctx context.Context, showtimeID string) ([]domain.Seat, error) {
	return nil, nil
}
func (m *mockShowtimeRepoForBooking) GetBookedSeatIDs(ctx context.Context, showtimeID string) ([]string, error) {
	return m.getBookedSeatsFn(ctx, showtimeID)
}

func TestCreateBooking_Success(t *testing.T) {
	showtimeRepo := &mockShowtimeRepoForBooking{
		findByIDFn: func(ctx context.Context, id string) (domain.Showtime, error) {
			return domain.Showtime{ID: id, HallID: "hall-1", StartTime: time.Now().Add(2 * time.Hour), Price: 12.50}, nil
		},
		getBookedSeatsFn: func(ctx context.Context, showtimeID string) ([]string, error) {
			return []string{}, nil
		},
	}
	bookingRepo := &mockBookingRepo{
		createFn: func(ctx context.Context, b domain.Booking, seatIDs []string) (domain.Booking, error) {
			b.ID = "booking-1"
			b.Status = "confirmed"
			return b, nil
		},
	}
	svc := service.NewBookingService(bookingRepo, showtimeRepo)
	booking, err := svc.Create(context.Background(), "user-1", "showtime-1", []string{"seat-1", "seat-2"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if booking.ID == "" {
		t.Fatal("expected booking ID, got empty")
	}
	if booking.TotalPrice != 25.00 {
		t.Fatalf("expected total price 25.00, got %v", booking.TotalPrice)
	}
}

func TestCreateBooking_PastShowtime(t *testing.T) {
	showtimeRepo := &mockShowtimeRepoForBooking{
		findByIDFn: func(ctx context.Context, id string) (domain.Showtime, error) {
			return domain.Showtime{ID: id, StartTime: time.Now().Add(-2 * time.Hour), Price: 12.50}, nil
		},
		getBookedSeatsFn: func(ctx context.Context, showtimeID string) ([]string, error) {
			return []string{}, nil
		},
	}
	svc := service.NewBookingService(&mockBookingRepo{}, showtimeRepo)
	_, err := svc.Create(context.Background(), "user-1", "showtime-1", []string{"seat-1"})
	if err == nil {
		t.Fatal("expected error for past showtime, got nil")
	}
}

func TestCreateBooking_SeatAlreadyBooked(t *testing.T) {
	showtimeRepo := &mockShowtimeRepoForBooking{
		findByIDFn: func(ctx context.Context, id string) (domain.Showtime, error) {
			return domain.Showtime{ID: id, StartTime: time.Now().Add(2 * time.Hour), Price: 12.50}, nil
		},
		getBookedSeatsFn: func(ctx context.Context, showtimeID string) ([]string, error) {
			return []string{"seat-1"}, nil
		},
	}
	svc := service.NewBookingService(&mockBookingRepo{}, showtimeRepo)
	_, err := svc.Create(context.Background(), "user-1", "showtime-1", []string{"seat-1"})
	if err == nil {
		t.Fatal("expected error for already-booked seat, got nil")
	}
}

func TestCreateBooking_ShowtimeNotFound(t *testing.T) {
	showtimeRepo := &mockShowtimeRepoForBooking{
		findByIDFn: func(ctx context.Context, id string) (domain.Showtime, error) {
			return domain.Showtime{}, sql.ErrNoRows
		},
		getBookedSeatsFn: func(ctx context.Context, showtimeID string) ([]string, error) {
			return []string{}, nil
		},
	}
	svc := service.NewBookingService(&mockBookingRepo{}, showtimeRepo)
	_, err := svc.Create(context.Background(), "user-1", "showtime-1", []string{"seat-1"})
	if err == nil {
		t.Fatal("expected error for missing showtime, got nil")
	}
}

func TestCancelBooking_NotOwner(t *testing.T) {
	bookingRepo := &mockBookingRepo{
		findByIDFn: func(ctx context.Context, id string) (domain.Booking, error) {
			return domain.Booking{ID: id, UserID: "other-user", Status: "confirmed"}, nil
		},
	}
	showtimeRepo := &mockShowtimeRepoForBooking{
		findByIDFn: func(ctx context.Context, id string) (domain.Showtime, error) {
			return domain.Showtime{StartTime: time.Now().Add(2 * time.Hour)}, nil
		},
	}
	svc := service.NewBookingService(bookingRepo, showtimeRepo)
	err := svc.Cancel(context.Background(), "requesting-user", "booking-1")
	if err == nil {
		t.Fatal("expected error when cancelling another user's booking, got nil")
	}
}
```

- [ ] **Step 2: Run tests — verify they fail**

```bash
go test ./internal/service/... -v -run TestCreateBooking
```

Expected: compile error — `service.NewBookingService` not defined yet.

- [ ] **Step 3: Implement `internal/service/booking.go`**

```go
package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aman-chandra314/cinema-booking-system/internal/domain"
)

type BookingRepo interface {
	Create(ctx context.Context, b domain.Booking, seatIDs []string) (domain.Booking, error)
	ListByUser(ctx context.Context, userID string) ([]domain.Booking, error)
	FindByID(ctx context.Context, id string) (domain.Booking, error)
	Cancel(ctx context.Context, id string) error
}

type BookingShowtimeRepo interface {
	List(ctx context.Context, movieID, date string) ([]domain.Showtime, error)
	FindByID(ctx context.Context, id string) (domain.Showtime, error)
	GetAvailableSeats(ctx context.Context, showtimeID string) ([]domain.Seat, error)
	GetBookedSeatIDs(ctx context.Context, showtimeID string) ([]string, error)
}

type BookingService struct {
	bookings  BookingRepo
	showtimes BookingShowtimeRepo
}

func NewBookingService(bookings BookingRepo, showtimes BookingShowtimeRepo) *BookingService {
	return &BookingService{bookings: bookings, showtimes: showtimes}
}

func (s *BookingService) Create(ctx context.Context, userID, showtimeID string, seatIDs []string) (domain.Booking, error) {
	if len(seatIDs) == 0 {
		return domain.Booking{}, errors.New("at least one seat must be selected")
	}

	showtime, err := s.showtimes.FindByID(ctx, showtimeID)
	if err != nil {
		return domain.Booking{}, fmt.Errorf("showtime not found: %w", err)
	}

	if showtime.StartTime.Before(time.Now()) {
		return domain.Booking{}, errors.New("cannot book a showtime that has already started")
	}

	bookedIDs, err := s.showtimes.GetBookedSeatIDs(ctx, showtimeID)
	if err != nil {
		return domain.Booking{}, fmt.Errorf("check booked seats: %w", err)
	}

	bookedSet := make(map[string]bool, len(bookedIDs))
	for _, id := range bookedIDs {
		bookedSet[id] = true
	}
	for _, id := range seatIDs {
		if bookedSet[id] {
			return domain.Booking{}, fmt.Errorf("seat %s is already booked", id)
		}
	}

	totalPrice := float64(len(seatIDs)) * showtime.Price
	return s.bookings.Create(ctx, domain.Booking{
		UserID:     userID,
		ShowtimeID: showtimeID,
		TotalPrice: totalPrice,
	}, seatIDs)
}

func (s *BookingService) List(ctx context.Context, userID string) ([]domain.Booking, error) {
	return s.bookings.ListByUser(ctx, userID)
}

func (s *BookingService) Get(ctx context.Context, userID, bookingID string) (domain.Booking, error) {
	b, err := s.bookings.FindByID(ctx, bookingID)
	if err != nil {
		return domain.Booking{}, fmt.Errorf("booking not found: %w", err)
	}
	if b.UserID != userID {
		return domain.Booking{}, errors.New("booking not found")
	}
	return b, nil
}

func (s *BookingService) Cancel(ctx context.Context, userID, bookingID string) error {
	b, err := s.bookings.FindByID(ctx, bookingID)
	if err != nil {
		return fmt.Errorf("booking not found: %w", err)
	}
	if b.UserID != userID {
		return errors.New("booking not found")
	}

	showtime, err := s.showtimes.FindByID(ctx, b.ShowtimeID)
	if err != nil {
		return fmt.Errorf("showtime not found: %w", err)
	}
	if showtime.StartTime.Before(time.Now()) {
		return errors.New("cannot cancel a booking after the showtime has started")
	}

	return s.bookings.Cancel(ctx, bookingID)
}
```

- [ ] **Step 4: Run all service tests — verify they pass**

```bash
go test ./internal/service/... -v
```

Expected: all tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/service/booking.go internal/service/booking_test.go
git commit -m "feat: add booking service with business rules and tests"
```

---

## Task 3: Booking Handler + Wire Routes

- [ ] **Step 1: Create `internal/handler/booking.go`**

```go
package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/aman-chandra314/cinema-booking-system/internal/domain"
	"github.com/aman-chandra314/cinema-booking-system/internal/middleware"
)

type BookingServiceIface interface {
	Create(ctx context.Context, userID, showtimeID string, seatIDs []string) (domain.Booking, error)
	List(ctx context.Context, userID string) ([]domain.Booking, error)
	Get(ctx context.Context, userID, bookingID string) (domain.Booking, error)
	Cancel(ctx context.Context, userID, bookingID string) error
}

type BookingHandler struct {
	bookings BookingServiceIface
}

func NewBookingHandler(bookings BookingServiceIface) *BookingHandler {
	return &BookingHandler{bookings: bookings}
}

func (h *BookingHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	var req struct {
		ShowtimeID string   `json:"showtime_id"`
		SeatIDs    []string `json:"seat_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.ShowtimeID == "" || len(req.SeatIDs) == 0 {
		writeError(w, http.StatusBadRequest, "showtime_id and seat_ids are required")
		return
	}
	booking, err := h.bookings.Create(r.Context(), userID, req.ShowtimeID, req.SeatIDs)
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, booking)
}

func (h *BookingHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	bookings, err := h.bookings.List(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if bookings == nil {
		bookings = []domain.Booking{}
	}
	writeJSON(w, http.StatusOK, bookings)
}

func (h *BookingHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	id := chi.URLParam(r, "id")
	booking, err := h.bookings.Get(r.Context(), userID, id)
	if err != nil {
		writeError(w, http.StatusNotFound, "booking not found")
		return
	}
	writeJSON(w, http.StatusOK, booking)
}

func (h *BookingHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	id := chi.URLParam(r, "id")
	if err := h.bookings.Cancel(r.Context(), userID, id); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
}
```

- [ ] **Step 2: Update `cmd/api/main.go` — final wiring with all routes**

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
	"github.com/aman-chandra314/cinema-booking-system/internal/middleware"
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
	bookingRepo := repository.NewBookingRepository(database)

	authSvc := service.NewAuthService(userRepo, cfg.JWTSecret)
	movieSvc := service.NewMovieService(movieRepo)
	cinemaSvc := service.NewCinemaService(cinemaRepo)
	showtimeSvc := service.NewShowtimeService(showtimeRepo)
	bookingSvc := service.NewBookingService(bookingRepo, showtimeRepo)

	authHandler := handler.NewAuthHandler(authSvc)
	movieHandler := handler.NewMovieHandler(movieSvc)
	cinemaHandler := handler.NewCinemaHandler(cinemaSvc)
	showtimeHandler := handler.NewShowtimeHandler(showtimeSvc)
	bookingHandler := handler.NewBookingHandler(bookingSvc)

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

		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(authSvc))
			r.Post("/bookings", bookingHandler.Create)
			r.Get("/bookings", bookingHandler.List)
			r.Get("/bookings/{id}", bookingHandler.Get)
			r.Delete("/bookings/{id}", bookingHandler.Cancel)
		})
	})

	log.Printf("server starting on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
```

- [ ] **Step 3: Run all tests**

```bash
go test ./... -v
```

Expected: all tests PASS.

- [ ] **Step 4: End-to-end smoke test**

```bash
go run ./cmd/api &

# Register and get token
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"name":"Bob","email":"bob@example.com","password":"pass1234"}' | jq -r .token)

# Get showtime and seat
SHOWTIME_ID=$(curl -s http://localhost:8080/api/v1/showtimes | jq -r '.[0].id')
SEAT_ID=$(curl -s http://localhost:8080/api/v1/showtimes/$SHOWTIME_ID/seats | jq -r '.[0].id')

# Create booking
curl -s -X POST http://localhost:8080/api/v1/bookings \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"showtime_id\":\"$SHOWTIME_ID\",\"seat_ids\":[\"$SEAT_ID\"]}" | jq .
```

Expected: `{"status": "confirmed", "total_price": 12.5, ...}`

```bash
# Try booking same seat again — should fail
curl -s -X POST http://localhost:8080/api/v1/bookings \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"showtime_id\":\"$SHOWTIME_ID\",\"seat_ids\":[\"$SEAT_ID\"]}" | jq .
```

Expected: `{"error": "seat ... is already booked"}`

```bash
# List bookings
curl -s http://localhost:8080/api/v1/bookings -H "Authorization: Bearer $TOKEN" | jq .
```

Expected: 1 booking in the array.

```bash
# Cancel booking
BOOKING_ID=$(curl -s http://localhost:8080/api/v1/bookings -H "Authorization: Bearer $TOKEN" | jq -r '.[0].id')
curl -s -X DELETE http://localhost:8080/api/v1/bookings/$BOOKING_ID \
  -H "Authorization: Bearer $TOKEN" | jq .
```

Expected: `{"status": "cancelled"}`

- [ ] **Step 5: Final commit**

```bash
git add internal/handler/booking.go cmd/api/main.go
git commit -m "feat: bookings complete — all routes wired, MVP done"
```
