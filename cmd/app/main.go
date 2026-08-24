package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"fx-gateway/internal/infrastructure/currency"
	"fx-gateway/internal/infrastructure/database"
	"fx-gateway/internal/infrastructure/handler"
	"fx-gateway/internal/infrastructure/repository"
	"fx-gateway/internal/usecase"

	"github.com/ilyakaznacheev/cleanenv"
	_ "github.com/jackc/pgx/v5"
)

func main() {
	///////////////////////////////////////////////////////
	// Create log file
	///////////////////////////////////////////////////////
	file, err := os.OpenFile("app.log", os.O_CREATE| os.O_WRONLY| os.O_APPEND, 0666)
	if err != nil {
		panic("Don't getting open file log" + err.Error())
	}
	defer file.Close()
	logOptions := &slog.HandlerOptions{
		Level:     slog.LevelDebug, 
		AddSource: true,           
	}
	fileHandler  := slog.NewJSONHandler(file, logOptions)
	// consoleHandler  := slog.NewTextHandler(os.Stdout, logOptions)
	logger := slog.New(fileHandler)
	slog.SetDefault(logger)

	///////////////////////////////////////////////////////
	// init Config
	///////////////////////////////////////////////////////
	var cfg database.AppConfig
	if err := cleanenv.ReadConfig(".env", &cfg); err != nil {
		slog.Error("Failed to read configuration", "error", err)
		os.Exit(1)
	}
	///////////////////////////////////////////////////////
	// init POSTGRES
	///////////////////////////////////////////////////////
	if cfg.DB.Password == "" || cfg.DB.DBHost == "" {
		slog.Warn("Critical database environment variables are missing")
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	db, err := database.InitDB(ctx, cfg.DB)
	if err != nil {
		slog.Error("Failed to initialize database", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	///////////////////////////////////////////////////////
	// Init REDIS
	///////////////////////////////////////////////////////
	if cfg.Redis.Address == "" {
		slog.Warn("Critical redis environment variables are missing")
	}

	rdb, err := database.InitRedis(ctx, cfg.Redis)
	if err != nil {
		slog.Error("Failed to initialize redis", "error", err)
		os.Exit(1)
	}
	defer rdb.Close()
	///////////////////////////////////////////////////////
	// Init clients
	///////////////////////////////////////////////////////
	txRepository := repository.NewPostgresTxRepository(db)
	if cfg.Provider.BaseURL == "" {
		slog.Warn("FX_PROVIDER_URL environment variable is missing, using default fallback")
	}
	apiClient := currency.NewAPIClient(cfg.Provider.BaseURL)
	cachedFetcher := currency.NewCachedCurrencyFetcher(rdb,apiClient)
	uCase := usecase.NewFXGatewayUseCase(cachedFetcher, txRepository, 150)
	/// Start worker for update vallute
	var wg sync.WaitGroup
	currencyWorker := currency.NewCurrencyWorker(uCase, 1*time.Hour)
	currencyWorker.Start(ctx, &wg)
	valuteHandler := handler.NewValuteHandler(uCase)
	///////////////////////////////////////////////////////
	// Marhutization http
	///////////////////////////////////////////////////////
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/price", valuteHandler.GetPrice)
	srv := &http.Server{
		Addr: ":8080",
		Handler: mux,
	}
	go func(){
		slog.Info("Server start on port :8080")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err,http.ErrServerClosed){
			slog.Error("Server forced to shutdown", "error", err)
			os.Exit(1)
		}

	}()
	slog.Info("Database successfully connected and pool configured!")
	///////////////////////////////////////////////////////
	// Graceful Shutdown
	///////////////////////////////////////////////////////
	<-ctx.Done() 
	slog.Info("Gotten signal quit. Starting stop program")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil{
		slog.Error("Server Shutdown Failed", "error", err)
	}
	slog.Info("Application has terminated successfully")
	wg.Wait()
}

