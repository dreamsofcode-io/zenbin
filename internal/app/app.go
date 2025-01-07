package app

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dreamsofcode-io/zenbin/internal/database"
	"github.com/dreamsofcode-io/zenbin/internal/middleware"
	"github.com/dreamsofcode-io/zenbin/internal/service/realip"
	"github.com/dreamsofcode-io/zenbin/internal/util/flash"
)

// App contains all of the application dependencies for the project.
type App struct {
	config     Config
	files      fs.FS
	logger     *slog.Logger
	db         *pgxpool.Pool
	ipresolver *realip.Service
}

func must[T any](x T, err error) T {
	if err != nil {
		panic(err)
	}

	return x
}

// New creates a new instance of the application.
func New(logger *slog.Logger, config Config, files fs.FS) (*App, error) {
	return &App{
		config:     config,
		logger:     logger,
		files:      files,
		ipresolver: realip.New(realip.LastXFFIPResolver),
	}, nil
}

// Start is used to start the application. The application
// will run until either the given context is cancelled, or
// the application is ended.
func (a *App) Start(ctx context.Context) error {
	db, err := database.Connect(ctx, a.logger, a.files)
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}

	a.db = db

	router, err := a.loadRoutes()
	if err != nil {
		return fmt.Errorf("failed when loading routes: %w", err)
	}

	middlewares := middleware.Chain(
		a.ipresolver.Middleware(),
		middleware.Logging(a.logger),
		flash.Middleware,
	)

	port := 3000
	srv := &http.Server{
		Addr:           fmt.Sprintf(":%d", port),
		Handler:        middlewares(router),
		MaxHeaderBytes: 1 << 20, // Max header size (e.g., 1 MB)
	}

	errCh := make(chan error, 1)

	go func() {
		err := srv.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("failed to listen and serve: %w", err)
		}

		close(errCh)
	}()

	a.logger.Info("server running", slog.Int("port", port))

	select {
	// Wait until we receive SIGINT (ctrl+c on cli)
	case <-ctx.Done():
		break
	case err := <-errCh:
		return err
	}

	sCtx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	srv.Shutdown(sCtx)

	return nil
}
