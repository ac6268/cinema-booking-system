# Cinema Booking System — Design Spec

**Date:** 2026-04-28  
**Status:** Approved

---

## Overview

A REST API backend for a cinema booking system built in Go. The system allows users to register, browse movies and showtimes, select seats, and make bookings. Designed as a learning project with real-world patterns: layered architecture, PostgreSQL, raw `database/sql`, JWT auth, Docker Compose, and SQL migrations.

---

## Goals

- Learn idiomatic Go backend development
- Understand layered architecture (Handler → Service → Repository)
- Practice raw SQL with `database/sql` (no ORM)
- Implement JWT-based authentication
- Use Docker Compose for local PostgreSQL
- Use `golang-migrate` for schema migrations

---

## Tech Stack

| Concern | Choice |
|---|---|
| Language | Go |
| API style | REST (JSON over HTTP) |
| Router | `chi` |
| Auth | JWT (HS256) via `golang-jwt/jwt` |
| Database | PostgreSQL |
| DB access | `database/sql` (standard library) |
| Migrations | `golang-migrate` |
| Password hashing | `bcrypt` |
| Config | `godotenv` + environment variables |
| Local infra | Docker Compose |

---

## Data Model

### `users`
| Column | Type | Notes |
|---|---|---|
| id | UUID | primary key |
| email | VARCHAR | unique, not null |
| password_hash | VARCHAR | bcrypt hash |
| name | VARCHAR | not null |
| created_at | TIMESTAMP | default now() |

### `cinemas`
| Column | Type | Notes |
|---|---|---|
| id | UUID | primary key |
| name | VARCHAR | not null |
| location | VARCHAR | not null |
| created_at | TIMESTAMP | default now() |

### `halls`
| Column | Type | Notes |
|---|---|---|
| id | UUID | primary key |
| cinema_id | UUID | FK → cinemas.id |
| name | VARCHAR | not null |
| total_seats | INT | not null |

### `seats`
| Column | Type | Notes |
|---|---|---|
| id | UUID | primary key |
| hall_id | UUID | FK → halls.id |
| row | VARCHAR | e.g. "A", "B" |
| number | INT | seat number within row |

### `movies`
| Column | Type | Notes |
|---|---|---|
| id | UUID | primary key |
| title | VARCHAR | not null |
| description | TEXT | |
| duration_minutes | INT | not null |
| genre | VARCHAR | |
| release_date | DATE | |

### `showtimes`
| Column | Type | Notes |
|---|---|---|
| id | UUID | primary key |
| movie_id | UUID | FK → movies.id |
| hall_id | UUID | FK → halls.id |
| start_time | TIMESTAMP | not null |
| end_time | TIMESTAMP | not null |
| price | NUMERIC(10,2) | not null |

### `bookings`
| Column | Type | Notes |
|---|---|---|
| id | UUID | primary key |
| user_id | UUID | FK → users.id |
| showtime_id | UUID | FK → showtimes.id |
| total_price | NUMERIC(10,2) | not null |
| status | VARCHAR | `confirmed` or `cancelled` |
| created_at | TIMESTAMP | default now() |

### `booking_seats`
| Column | Type | Notes |
|---|---|---|
| booking_id | UUID | FK → bookings.id, composite PK |
| seat_id | UUID | FK → seats.id, composite PK |

**Key relationships:**
- A cinema has many halls; a hall has many seats
- A showtime links a movie to a hall at a specific time
- A booking links a user to a showtime; `booking_seats` tracks which specific seats are reserved
- Seat availability = seats in the hall minus seats already in `booking_seats` for that showtime

---

## API Endpoints

### Auth (public)
```
POST /api/v1/auth/register    — register (name, email, password) → user + JWT
POST /api/v1/auth/login       — login (email, password) → JWT
```

### Movies (public)
```
GET /api/v1/movies            — list all movies
GET /api/v1/movies/:id        — get movie details
```

### Cinemas (public)
```
GET /api/v1/cinemas           — list all cinemas
GET /api/v1/cinemas/:id       — get cinema with its halls
```

### Showtimes (public)
```
GET /api/v1/showtimes                  — list showtimes (filter: ?movie_id=, ?date=)
GET /api/v1/showtimes/:id              — get showtime details
GET /api/v1/showtimes/:id/seats        — get available seats for a showtime
```

### Bookings (protected — JWT required)
```
POST   /api/v1/bookings               — create booking (showtime_id, seat_ids[])
GET    /api/v1/bookings               — get current user's booking history
GET    /api/v1/bookings/:id           — get a specific booking
DELETE /api/v1/bookings/:id           — cancel a booking
```

JWT passed as `Authorization: Bearer <token>` header.

---

## Project Structure

```
cinema-booking-system/
├── cmd/
│   └── api/
│       └── main.go              ← entry point, wires everything together
├── internal/
│   ├── domain/                  ← shared structs (User, Movie, Booking, etc.)
│   ├── handler/                 ← HTTP handlers, request parsing, response writing
│   │   ├── auth.go
│   │   ├── movie.go
│   │   ├── cinema.go
│   │   ├── showtime.go
│   │   └── booking.go
│   ├── service/                 ← business logic
│   │   ├── auth.go
│   │   ├── movie.go
│   │   ├── cinema.go
│   │   ├── showtime.go
│   │   └── booking.go
│   ├── repository/              ← raw SQL queries via database/sql
│   │   ├── user.go
│   │   ├── movie.go
│   │   ├── cinema.go
│   │   ├── showtime.go
│   │   └── booking.go
│   ├── middleware/
│   │   └── auth.go              ← JWT validation, injects user_id into context
│   └── db/
│       └── db.go                ← PostgreSQL connection setup
├── migrations/
│   ├── 000001_init_schema.up.sql
│   └── 000001_init_schema.down.sql
├── configs/
│   └── config.go                ← loads env vars (DB_URL, JWT_SECRET, PORT)
├── docker-compose.yml           ← PostgreSQL service
├── .env.example
├── Makefile                     ← make run, make migrate, make test
└── go.mod
```

---

## Request Flow

Each request passes through layers with a single direction of dependency:

```
HTTP Request
    ↓
[Middleware] — validates JWT, injects user_id into context
    ↓
[Handler] — parses + validates request body/params, calls service
    ↓
[Service] — enforces business rules, calls repository
    ↓
[Repository] — executes SQL, returns domain types
    ↓
[Handler] — writes JSON response
```

### Example: Create Booking

1. Middleware validates JWT, extracts `user_id` → request context
2. Handler parses `{ showtime_id, seat_ids[] }`, validates non-empty
3. Service:
   - Verifies showtime exists and `start_time` is in the future
   - Verifies all seat IDs belong to the showtime's hall
   - Verifies none of the seats are already booked for this showtime
   - Calculates `total_price = len(seat_ids) × showtime.price`
   - Creates `booking` + `booking_seats` rows in a single DB transaction
4. Handler returns `201 Created` with booking details

---

## Business Rules

- Seat conflict check runs inside a DB transaction to prevent double-booking under concurrent requests
- Cancellation only allowed if the showtime has not yet started
- A user cannot book the same seat twice for the same showtime
- Passwords are hashed with bcrypt before storage; plaintext is never stored

---

## Error Handling

| Code | Meaning |
|---|---|
| 400 | Invalid input (missing/malformed fields) |
| 401 | Missing or invalid JWT |
| 404 | Resource not found |
| 409 | Seat already booked (conflict) |
| 500 | Unexpected server error (logged server-side, generic message to client) |

---

## Testing Strategy

**Service layer (unit tests):** Mock the repository interface, test business logic in isolation.
- Happy path: booking available seats
- Sad paths: already-booked seats, past showtime, invalid seat IDs

**Repository layer (integration tests):** Run against a real PostgreSQL instance via Docker.
- Verify queries return correct data
- Verify transaction rollback on conflict

**Interfaces enable testability:** Each service depends on a repository interface, not a concrete type. In tests, inject a mock. In production, inject the real DB implementation.

```go
type BookingRepository interface {
    GetShowtime(ctx context.Context, id string) (domain.Showtime, error)
    GetSeatsByIDs(ctx context.Context, ids []string) ([]domain.Seat, error)
    CreateBooking(ctx context.Context, b domain.Booking, seatIDs []string) (domain.Booking, error)
    GetBookedSeatIDs(ctx context.Context, showtimeID string) ([]string, error)
}
```

---

## Out of Scope

- Admin API (manage movies, halls, showtimes) — good stretch goal after core is working
- Payment integration
- Email notifications
- gRPC
- JWT refresh tokens
