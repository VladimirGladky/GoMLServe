package repository

import (
	"GoMLServe/internal/domain/models"
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrUserNotFound = errors.New("user not found")
	ErrUserExists   = errors.New("user already exists")
)

type MLRepositoryInterface interface {
	SaveUser(user *models.User) error
	User(email string) (*models.User, error)
}

type MLRepository struct {
	db  *pgx.Conn
	ctx context.Context
}

func NewMLRepository(ctx context.Context, db *pgx.Conn) *MLRepository {
	return &MLRepository{
		db:  db,
		ctx: ctx,
	}
}

func (r *MLRepository) SaveUser(user *models.User) error {
	_, err := r.db.Exec(r.ctx, "INSERT INTO users (email, pass_hash) VALUES ($1, $2)", user.Email, user.PassHash)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return fmt.Errorf("%w: %s", ErrUserExists, user.Email)
		}
		return fmt.Errorf("failed to save user: %w", err)
	}
	return nil
}

func (r *MLRepository) User(email string) (*models.User, error) {
	var user models.User
	err := r.db.QueryRow(r.ctx,
		"SELECT email, pass_hash FROM users WHERE email = $1",
		email).Scan(&user.Email, &user.PassHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &user, nil
}
