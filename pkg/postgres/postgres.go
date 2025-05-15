package postgres

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
)

type Config struct {
	Host     string `yaml:"postgres_host" env:"POSTGRES_HOST" env-default:"postgres"`
	Port     string `yaml:"postgres_port" env:"POSTGRES_PORT" env-default:"5432"`
	Database string `yaml:"postgres_db" env:"POSTGRES_DB" env-default:"postgres"`
	User     string `yaml:"postgres_user" env:"POSTGRES_USER" env-default:"root"`
	Password string `yaml:"postgres_password" env:"POSTGRES_PASSWORD" env-default:"1234"`
}

func New(config Config) (*pgx.Conn, error) {
	connString := fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
		config.User,
		config.Password,
		config.Host,
		config.Port,
		config.Database)
	conn, err := pgx.Connect(context.Background(), connString)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %w", err)
	}
	return conn, nil
}
