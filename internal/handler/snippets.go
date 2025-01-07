package handler

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dreamsofcode-io/zenbin/internal/component"
	"github.com/dreamsofcode-io/zenbin/internal/repository"
	"github.com/dreamsofcode-io/zenbin/internal/service/realip"
	"github.com/dreamsofcode-io/zenbin/internal/util/flash"
	"github.com/dreamsofcode-io/zenbin/internal/util/shortid"
)

type Handler struct {
	logger     *slog.Logger
	repo       *repository.Queries
	db         *pgxpool.Pool
	ipResolver *realip.Service
}

func New(
	logger *slog.Logger,
	repo *repository.Queries,
	db *pgxpool.Pool,
	ipService *realip.Service,
) *Handler {
	return &Handler{
		logger:     logger,
		repo:       repo,
		db:         db,
		ipResolver: ipService,
	}
}

// 1MB maxBodySize
const maxBodySize = 1 << 20

func (h *Handler) CreateSnippet(w http.ResponseWriter, r *http.Request) {
	// Limit the size of the request body
	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)

	content := r.FormValue("content")
	if content == "" {
		flash.SetFlashMessage(w, "error", "content cannot be empty")
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	ctx := r.Context()
	ip := h.ipResolver.RealIPForRequest(r)
	id, err := uuid.NewV7()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	count, err := h.repo.CheckSnippet24Hours(ctx, ip)
	if err != nil {
		h.logger.Error("failed to check snippet", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	const maxCount = 5
	if count >= maxCount {
		flash.SetFlashMessage(w, "error", "Snippets exceeded for the day, try again tomorrow")
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	snippet, err := h.repo.InsertSnippetCheck(ctx, repository.InsertSnippetCheckParams{
		ID:      id,
		Content: content,
		Ip:      ip,
	})
	if err != nil {
		h.logger.Error("failed to insert snippet", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	host := r.Host
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}

	uri := fmt.Sprintf("%s://%s/%s", scheme, host, shortid.GetShortID(snippet.ID))

	fmt.Println(uri)

	http.Redirect(w, r, uri, http.StatusFound)
}

func (h *Handler) GetSnippet(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	shortID := r.PathValue("id")

	id, err := shortid.GetLongID(shortID)
	if err != nil {
		h.logger.Error("failed to get long id", slog.Any("error", err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	snippet, err := h.repo.FindSnippetByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		w.WriteHeader(http.StatusNotFound)
		component.NotFound().Render(ctx, w)
		return
	}
	if err != nil {
		return
	}

	host := r.Host
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}

	uri := fmt.Sprintf("%s://%s/%s", scheme, host, shortid.GetShortID(snippet.ID))

	component.SnippetPage(snippet, uri).Render(ctx, w)
}
