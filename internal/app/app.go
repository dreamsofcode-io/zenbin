package app

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/dreamsofcode-io/zenbin/internal/middleware"
	"github.com/dreamsofcode-io/zenbin/internal/service/realip"
	"github.com/dreamsofcode-io/zenbin/internal/util/flash"
)

// App contains all of the application dependencies for the project.
type App struct {
	config     Config
	files      fs.FS
	logger     *slog.Logger
	rdb        *redis.Client
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
	redisURL, ok := os.LookupEnv("REDIS_URL")
	if !ok {
		return nil, fmt.Errorf("Must set redis URL")
	}

	cfg, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse redis url: %w", err)
	}

	rdb := redis.NewClient(cfg)

	return &App{
		config:     config,
		logger:     logger,
		files:      files,
		rdb:        rdb,
		ipresolver: realip.New(realip.LastXFFIPResolver),
	}, nil
}

// Start is used to start the application. The application
// will run until either the given context is cancelled, or
// the application is ended.
func (a *App) Start(ctx context.Context) error {
	if err := a.rdb.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("ping redis: %w", err)
	}

	router, err := a.loadRoutes()
	if err != nil {
		return fmt.Errorf("failed when loading routes: %w", err)
	}

	middlewares := middleware.Chain(
		a.ipresolver.Middleware(),
		middleware.Logging(a.logger),
		flash.Middleware,
	)

	port := getPort(3000)
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

func getPort(defaultPort int) int {
	portStr := os.Getenv("PORT")
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return defaultPort
	}
	return port
}
