package apphttp

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/drobyshevv/url-shortening-service/internal/config"
	"github.com/drobyshevv/url-shortening-service/internal/handler"
	"github.com/drobyshevv/url-shortening-service/internal/repository/postgres"
	"github.com/drobyshevv/url-shortening-service/internal/service"
	"github.com/jackc/pgx/v5/pgxpool"
)

type App struct {
	server *http.Server
	pool   *pgxpool.Pool
}

func NewApp(log *slog.Logger, cfg *config.Config) (*App, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	pool, err := postgres.NewPool(ctx, config.ConvertToStringDSN(cfg.DB))
	if err != nil {
		log.Error("failed to init db", "error", err)
		return nil, err
	}
	r := postgres.NewUrlRepository(pool)
	s := service.NewServiceUrl(r)
	h := handler.NewUrlHandler(s, log)

	mux := http.NewServeMux()

	mux.HandleFunc("POST /shorten", h.PostUrl)
	mux.HandleFunc("GET /shorten/{code}", h.GetUrl)
	mux.HandleFunc("PUT /shorten/{code}", h.PutUrl)
	mux.HandleFunc("DELETE /shorten/{code}", h.DeleteUrl)
	mux.HandleFunc("GET /shorten/{code}/stats", h.GetUrlStatistics)
	mux.HandleFunc("GET /shorten/{code}/redirect", h.RedirectUrl)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Srv.Port),
		Handler: mux,
	}

	return &App{
		server: server,
		pool:   pool,
	}, nil
}

func (a *App) Run() error {
	return a.server.ListenAndServe()
}

func (a *App) Stop(ctx context.Context) error {
	if err := a.server.Shutdown(ctx); err != nil {
		return err
	}
	a.pool.Close()
	return nil
}

func (a *App) Close() {
	a.server.Close()
	a.pool.Close()
}
