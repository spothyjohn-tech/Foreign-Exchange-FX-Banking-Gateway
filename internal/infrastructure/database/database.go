package database

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"time"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type DBConfig struct{
	DBName   string `env:"DB_NAME" env-required:"true"`
	DBHost   string `env:"DB_HOST" env-required:"true"`
	DBPort   string `env:"DB_PORT" env-default:"5432"`
	Password string `env:"DB_PASSWORD" env-required:"true"`
	User     string `env:"DB_USER" env-required:"true"`
	SSLMode  string `env:"DB_SSL_MODE" env-default:"disable"`
}

type ProviderConfig struct {
	BaseURL string `env:"FX_PROVIDER_URL" env-required:"true"`
}

type AppConfig struct {
	DB       DBConfig
	Redis    RedisConfig
	Provider ProviderConfig
}

func InitDB(ctx context.Context,cfg DBConfig) (*sql.DB, error){
	query := url.Values{}
	query.Set("sslmode",cfg.SSLMode)

	u := &url.URL{
		Scheme: "postgres",
		User: url.UserPassword(cfg.User,cfg.Password),
		Host: fmt.Sprintf("%s:%s", cfg.DBHost, cfg.DBPort),
		Path: cfg.DBName,
		RawQuery: query.Encode(),	
	}
	
	db, err := sql.Open("pgx", u.String())
	if err != nil{
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5*time.Minute)
	db.SetConnMaxIdleTime(3*time.Minute)

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil{
		_ = db.Close()
		return nil, fmt.Errorf("database index ping failed: %w", err)
	}
	return db, nil
}