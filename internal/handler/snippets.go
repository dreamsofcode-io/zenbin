package handler

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/dreamsofcode-io/zenbin/internal/component"
	"github.com/dreamsofcode-io/zenbin/internal/service/realip"
	"github.com/dreamsofcode-io/zenbin/internal/util/flash"
	"github.com/dreamsofcode-io/zenbin/internal/util/shortid"
)

type Handler struct {
	logger     *slog.Logger
	rdb        *redis.Client
	ipResolver *realip.Service
}

func New(
	logger *slog.Logger,
	rdb *redis.Client,
	ipService *realip.Service,
) *Handler {
	return &Handler{
		logger:     logger,
		rdb:        rdb,
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

	const maxCount = 5

	countKey := fmt.Sprintf("counts:%s", time.Now().UTC().Truncate(time.Hour*24))

	p := h.rdb.Pipeline()
	incrRes := p.HIncrBy(ctx, countKey, ip, 1)
	p.ExpireXX(ctx, countKey, time.Hour*25)

	if _, err := p.Exec(ctx); err != nil {
		h.logger.Error("failed to get counts", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	count, err := incrRes.Result()
	if err != nil {
		h.logger.Error("failed to get counts", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if count >= maxCount {
		flash.SetFlashMessage(w, "error", "Snippets exceeded for the day, try again tomorrow")
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	p = h.rdb.Pipeline()
	p.HSet(ctx, id.String(), "ip", ip, "content", content, "created_at", time.Now().UTC())
	p.Expire(ctx, id.String(), time.Hour*24*7)

	if _, err := p.Exec(ctx); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	host := r.Host
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}

	uri := fmt.Sprintf("%s://%s/%s", scheme, host, shortid.GetShortID(id))

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

	content, err := h.rdb.HGet(ctx, id.String(), "content").Result()
	if errors.Is(err, redis.Nil) {
		w.WriteHeader(http.StatusNotFound)
		component.NotFound().Render(ctx, w)
		return
	} else if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	host := r.Host
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}

	uri := fmt.Sprintf("%s://%s/%s", scheme, host, shortID)

	component.SnippetPage(content, uri).Render(ctx, w)
}
