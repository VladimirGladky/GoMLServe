package config

import (
	"GoMLServe/pkg/postgres"
	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

type Config struct {
	Postgres postgres.Config `yaml:"Postgres" env:"POSTGRES"`
	Port     string          `yaml:"port" env:"PORT" env-default:"50051"`
	Host     string          `yaml:"host" env:"HOST" env-default:"localhost"`
}

func New() (*Config, error) {
	_ = godotenv.Load(".env")
	var cfg Config
	if err := cleanenv.ReadConfig("./config/config.yaml", &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
