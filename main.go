//go:generate npx @tailwindcss/cli -i ./static/css/input.css -o ./static/css/style.css --minify
//go:generate templ generate
package main

import (
	"context"
	"embed"
	"log/slog"
	"os"
	"os/signal"

	"github.com/joho/godotenv"

	"github.com/dreamsofcode-io/zenbin/internal/app"
)

//go:embed static
var files embed.FS

func main() {
	godotenv.Load()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	app, err := app.New(logger, app.Config{}, files)
	if err != nil {
		logger.Error("failed to create app", slog.Any("error", err))
	}

	if err := app.Start(ctx); err != nil {
		logger.Error("failed to start app", slog.Any("error", err))
	}
}
