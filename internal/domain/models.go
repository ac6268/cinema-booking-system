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
