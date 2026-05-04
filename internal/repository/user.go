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
	//write query
	query := `
	    INSERT INTO users (email, password_hash, name)
		VALUES ($1, $2, $3)
		RETURNING id, email, password_hash, name, created_at
	  `
	//run the query
	row := r.db.QueryRowContext(ctx, query, u.Email, u.PasswordHash, u.Name)

	//read the returned value
	if err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.CreatedAt); err != nil {
		return domain.User{}, fmt.Errorf("create user: %w", err)
	}

	return u, nil

}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (domain.User, error) {
	// write query
	query := `SELECT id, email, password_hash, name, created_at FROM users WHERE email = $1`

	// run the query
	row := r.db.QueryRowContext(ctx, query, email)

	var u domain.User

	//scan the result
	if err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.CreatedAt); err != nil {
		return domain.User{}, fmt.Errorf("find user by email: %w", err)
	}

	return u, nil

}

func (r *UserRepository) FindByID(ctx context.Context, id string) (domain.User, error) {
	//write query
	query := `
	    SELECT  id , email, password_hash, name, created_at FROM users WHERE id = $1
	 `
	// run query
	row := r.db.QueryRowContext(ctx, query, id)

	var u domain.User

	//scan query
	if err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.CreatedAt); err != nil {
		return domain.User{}, fmt.Errorf("find user by id: %w", err)
	}

	return u, nil

}
