package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/KyryloY/demand-pricing-go/internal/app"
	"github.com/KyryloY/demand-pricing-go/internal/repository"
	"github.com/KyryloY/demand-pricing-go/internal/service"
	httptransport "github.com/KyryloY/demand-pricing-go/internal/transport/http"
)

func main() {
	config := app.LoadConfig()
	pool, err := pgxpool.New(context.Background(), config.DatabaseURL)
	if err != nil {
		fatalf("create database pool: %v", err)
	}
	defer pool.Close()
	if err := pool.Ping(context.Background()); err != nil {
		fatalf("ping database: %v", err)
	}
	if err := app.RunMigrations(config.DatabaseURL, config.MigrationsPath); err != nil {
		fatalf("run migrations: %v", err)
	}

	catalogRepository := repository.NewCatalogRepository(pool)
	salesRepository := repository.NewSalesRepository(pool)
	forecastRepository := repository.NewForecastRepository(pool)
	recommendationRepository := repository.NewRecommendationRepository(pool)
	server := &http.Server{
		Addr: config.HTTPAddr,
		Handler: httptransport.NewRouter(httptransport.Dependencies{
			Catalog:        catalogRepository,
			Importer:       service.NewSalesImporter(salesRepository),
			Forecast:       service.NewForecastService(forecastRepository),
			Recommendation: service.NewRecommendationService(recommendationRepository),
			Ready:          pool.Ping,
		}),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fatalf("serve API: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownContext); err != nil {
		fatalf("shutdown API: %v", err)
	}
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
