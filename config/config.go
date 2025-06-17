package config

import (
	"log"
	"sync"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Config struct {
	Logger   Logger
	App      AppConfig
	Server   ServerConfig
	Postgres PostgresConfig
	Service  ServiceConfig
}

type PostgresConfig struct {
	DataSource        string `env:"DB_DATA_SOURCE,required"`
	PathToMigrations  string `env:"DB_PATH_TO_MIGRATION,required"`
	ApplicationSchema string `env:"DB_APPLICATION_SCHEMA,required"`
}

type Logger struct {
	LogLevel    string `env:"LOG_LEVEL,required"`
	ServiceName string `env:"LOG_SERVICE_NAME,required"`
	LogPath     string `env:"LOG_PATH"`
}

type AppConfig struct {
	ShutdownTimeout time.Duration `env:"APP_SHUTDOWN_TIMEOUT,required"`
}

type ServerConfig struct {
	JwtKey        string        `env:"SERVER_JWT_KEY,required"`
	Addr          string        `env:"SERVER_ADDR,required"`
	TokenLifeTime time.Duration `env:"SERVER_TOKEN_LIFE_TIME,required"`
	LogQuerys     bool          `env:"SERVER_LOG_QUERYS"`
}

type ServiceConfig struct {
	NotificationPeriod time.Duration `env:"SERVICE_NOTIFICATION_PERIOD,required"`
}

var (
	config Config
	once   sync.Once
)

func Get() *Config {
	once.Do(func() {
		_ = godotenv.Load()
		if err := env.Parse(&config); err != nil {
			log.Fatal(err)
		}
	})
	return &config
}
