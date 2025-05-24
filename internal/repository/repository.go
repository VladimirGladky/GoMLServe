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
	GetResult(email string, id string) (*models.Result, error)
	SavePhrase(id, email string, expr *models.Phrase) error
	UpdateResult(id, email string, result *models.Result) error
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

func (r *MLRepository) GetResult(email string, id string) (*models.Result, error) {
	var userId int
	err := r.db.QueryRow(r.ctx, "SELECT id FROM users WHERE email = $1", email).Scan(&userId)
	if err != nil {
		return nil, fmt.Errorf("failed to get user id: %w", err)
	}
	var result models.Result
	err = r.db.QueryRow(r.ctx,
		"SELECT res, expr FROM user_expressions WHERE id = $1 AND user_id = $2",
		id, userId).Scan(&result.Res, &result.Text)
	if err != nil {
		return nil, fmt.Errorf("failed to get result: %w", err)
	}
	return &result, nil
}

func (r *MLRepository) SavePhrase(id, email string, expr *models.Phrase) error {
	var userId int
	err := r.db.QueryRow(r.ctx, "SELECT id FROM users WHERE email = $1", email).Scan(&userId)
	if err != nil {
		return fmt.Errorf("failed to get user id: %w", err)
	}
	_, err = r.db.Exec(r.ctx,
		"INSERT INTO user_expressions (id,user_id, expr, res) VALUES ($1, $2, $3,$4)",
		id, userId, expr.Text, "")
	if err != nil {
		return fmt.Errorf("failed to save phrase: %w", err)
	}
	return nil
}

func (r *MLRepository) UpdateResult(id, email string, result *models.Result) error {
	var userId int
	err := r.db.QueryRow(r.ctx, "SELECT id FROM users WHERE email = $1", email).Scan(&userId)
	if err != nil {
		return fmt.Errorf("failed to get user id: %w", err)
	}
	_, err = r.db.Exec(r.ctx,
		"UPDATE user_expressions SET res = $1 WHERE id = $2 AND user_id = $3",
		result.Res, id, userId)
	if err != nil {
		return fmt.Errorf("failed to update result: %w", err)
	}
	return nil
}
