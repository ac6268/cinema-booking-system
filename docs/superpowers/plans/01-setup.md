# Plan 1: Project Setup & Infrastructure

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Bootstrap the Go project with module setup, config loading, DB connection, Docker Compose, migrations, and domain models.

**Architecture:** Single binary (`cmd/api`). Config loaded from env vars via `godotenv`. PostgreSQL connection via `database/sql`. Migrations via `golang-migrate`.

**Tech Stack:** Go, `godotenv`, `golang-migrate`, `lib/pq`, PostgreSQL, Docker Compose.

---

## Files Created

```
go.mod
.env.example
docker-compose.yml
Makefile
configs/config.go
internal/db/db.go
internal/domain/models.go
migrations/000001_init_schema.up.sql
migrations/000001_init_schema.down.sql
cmd/api/main.go   (skeleton)
```

---

## Task 1: Go Module + Dependencies

- [ ] **Step 1: Initialise module**

```bash
cd /Users/a548717/cinema-booking-system
go mod init github.com/aman-chandra314/cinema-booking-system
```

Expected: `go.mod` created.

- [ ] **Step 2: Install dependencies**

```bash
go get github.com/go-chi/chi/v5
go get github.com/golang-jwt/jwt/v5
go get golang.org/x/crypto
go get github.com/joho/godotenv
go get github.com/golang-migrate/migrate/v4
go get github.com/golang-migrate/migrate/v4/database/postgres
go get github.com/golang-migrate/migrate/v4/source/file
go get github.com/lib/pq
```

Expected: `go.sum` created, all packages in `go.mod`.

- [ ] **Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "feat: initialise Go module and dependencies"
```

---

## Task 2: Config + Docker + Makefile

- [ ] **Step 1: Create `.env.example`**

```
DB_URL=postgres://postgres:postgres@localhost:5432/cinema_booking?sslmode=disable
JWT_SECRET=replace-with-a-long-random-secret
PORT=8080
```

- [ ] **Step 2: Create `docker-compose.yml`**

```yaml
version: "3.9"
services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
      POSTGRES_DB: cinema_booking
    ports:
      - "5432:5432"
    volumes:
      - pgdata:/var/lib/postgresql/data

volumes:
  pgdata:
```

- [ ] **Step 3: Create `Makefile`**

```makefile
.PHONY: run migrate-up migrate-down test

run:
	go run ./cmd/api

migrate-up:
	migrate -path migrations -database "$(DB_URL)" up

migrate-down:
	migrate -path migrations -database "$(DB_URL)" down

test:
	go test ./...
```

- [ ] **Step 4: Create `configs/config.go`**

```go
package configs

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DBURL     string
	JWTSecret string
	Port      string
}

func Load() Config {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, reading from environment")
	}
	return Config{
		DBURL:     mustGet("DB_URL"),
		JWTSecret: mustGet("JWT_SECRET"),
		Port:      getOrDefault("PORT", "8080"),
	}
}

func mustGet(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required env var %s is not set", key)
	}
	return v
}

func getOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
```

- [ ] **Step 5: Create `internal/db/db.go`**

```go
package db

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

func Connect(dsn string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}
	return db, nil
}
```

- [ ] **Step 6: Commit**

```bash
git add .env.example docker-compose.yml Makefile configs/ internal/db/
git commit -m "feat: add config, DB connection, docker-compose, Makefile"
```

---

## Task 3: Database Migrations

- [ ] **Step 1: Create `migrations/000001_init_schema.up.sql`**

```sql
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE cinemas (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    location VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE halls (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cinema_id UUID NOT NULL REFERENCES cinemas(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    total_seats INT NOT NULL
);

CREATE TABLE seats (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    hall_id UUID NOT NULL REFERENCES halls(id) ON DELETE CASCADE,
    row VARCHAR(10) NOT NULL,
    number INT NOT NULL
);

CREATE TABLE movies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title VARCHAR(255) NOT NULL,
    description TEXT,
    duration_minutes INT NOT NULL,
    genre VARCHAR(100),
    release_date DATE
);

CREATE TABLE showtimes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    movie_id UUID NOT NULL REFERENCES movies(id) ON DELETE CASCADE,
    hall_id UUID NOT NULL REFERENCES halls(id) ON DELETE CASCADE,
    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP NOT NULL,
    price NUMERIC(10,2) NOT NULL
);

CREATE TABLE bookings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    showtime_id UUID NOT NULL REFERENCES showtimes(id) ON DELETE CASCADE,
    total_price NUMERIC(10,2) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'confirmed',
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE booking_seats (
    booking_id UUID NOT NULL REFERENCES bookings(id) ON DELETE CASCADE,
    seat_id UUID NOT NULL REFERENCES seats(id) ON DELETE CASCADE,
    PRIMARY KEY (booking_id, seat_id)
);
```

- [ ] **Step 2: Create `migrations/000001_init_schema.down.sql`**

```sql
DROP TABLE IF EXISTS booking_seats;
DROP TABLE IF EXISTS bookings;
DROP TABLE IF EXISTS showtimes;
DROP TABLE IF EXISTS movies;
DROP TABLE IF EXISTS seats;
DROP TABLE IF EXISTS halls;
DROP TABLE IF EXISTS cinemas;
DROP TABLE IF EXISTS users;
```

- [ ] **Step 3: Start Docker and run migration**

```bash
docker compose up -d
cp .env.example .env
export $(cat .env | xargs)
make migrate-up
```

Expected: `1/u init_schema (Xms)`

- [ ] **Step 4: Verify tables exist**

```bash
docker exec -it $(docker ps -qf "name=postgres") psql -U postgres -d cinema_booking -c "\dt"
```

Expected: 8 tables listed.

- [ ] **Step 5: Commit**

```bash
git add migrations/
git commit -m "feat: add database schema migration"
```

---

## Task 4: Domain Models + Skeleton main.go

- [ ] **Step 1: Create `internal/domain/models.go`**

```go
package domain

import "time"

type User struct {
	ID           string
	Email        string
	PasswordHash string
	Name         string
	CreatedAt    time.Time
}

type Cinema struct {
	ID        string
	Name      string
	Location  string
	CreatedAt time.Time
	Halls     []Hall
}

type Hall struct {
	ID         string
	CinemaID   string
	Name       string
	TotalSeats int
}

type Seat struct {
	ID     string
	HallID string
	Row    string
	Number int
}

type Movie struct {
	ID              string
	Title           string
	Description     string
	DurationMinutes int
	Genre           string
	ReleaseDate     *time.Time
}

type Showtime struct {
	ID        string
	MovieID   string
	HallID    string
	StartTime time.Time
	EndTime   time.Time
	Price     float64
}

type Booking struct {
	ID         string
	UserID     string
	ShowtimeID string
	TotalPrice float64
	Status     string
	CreatedAt  time.Time
	Seats      []Seat
}
```

- [ ] **Step 2: Create skeleton `cmd/api/main.go`**

```go
package main

import (
	"log"
	"net/http"

	"github.com/aman-chandra314/cinema-booking-system/configs"
	"github.com/aman-chandra314/cinema-booking-system/internal/db"
)

func main() {
	cfg := configs.Load()

	database, err := db.Connect(cfg.DBURL)
	if err != nil {
		log.Fatalf("connect to db: %v", err)
	}
	defer database.Close()

	log.Printf("server starting on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, nil); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
```

- [ ] **Step 3: Verify compilation**

```bash
go build ./...
```

Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add internal/domain/ cmd/
git commit -m "feat: add domain models and skeleton main.go"
```
