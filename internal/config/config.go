package config

import (
	"GoMLServe/pkg/postgres"
	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Postgres        postgres.Config `yaml:"Postgres" env:"POSTGRES"`
	GRPCPort        string          `yaml:"grpc_port" env:"GRPC_PORT" env-default:"50051"`
	GRPCHost        string          `yaml:"grpc_host" env:"GRPC_HOST" env-default:"localhost"`
	GRPCGatewayPort string          `yaml:"grpc_gateway_port" env:"GRPC_GATEWAY_PORT" env-default:"8080"`
	GRPCGatewayHost string          `yaml:"grpc_gateway_host" env:"GRPC_GATEWAY_HOST" env-default:"localhost"`
}

func New() (*Config, error) {
	var cfg Config
	if err := cleanenv.ReadConfig("./config/config.yaml", &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
