package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/drobyshevv/url-shortening-service/internal/errs"
	"github.com/drobyshevv/url-shortening-service/internal/model"
	"github.com/go-playground/validator/v10"
)

type UrlServiceWriter interface {
	PostUrl(context.Context, string) (*model.ShortLink, error)
	PutUrl(context.Context, string, string) (*model.ShortLink, error)
	DeleteUrl(context.Context, string) error
}

type UrlServiceReader interface {
	GetUrl(context.Context, string) (*model.ShortLink, error)
	GetUrlStatistics(context.Context, string) (*model.UrlStats, error)
	GetAllUrlsWithStatistics(context.Context, int, int, model.UrlFilter) ([]model.UrlStats, error)
}

type UrlHandlerReader struct {
	service  UrlServiceReader
	validate *validator.Validate
	log      *slog.Logger
}

type UrlHandlerWriter struct {
	service  UrlServiceWriter
	validate *validator.Validate
	log      *slog.Logger
}

func NewUrlHandlerReader(service UrlServiceReader, log *slog.Logger) *UrlHandlerReader {
	return &UrlHandlerReader{
		service:  service,
		validate: validator.New(),
		log:      log,
	}
}

func NewUrlHandlerWriter(service UrlServiceWriter, log *slog.Logger) *UrlHandlerWriter {
	return &UrlHandlerWriter{
		service:  service,
		validate: validator.New(),
		log:      log,
	}
}

type CreateOrUpdateURLRequest struct {
	Url string `json:"url" validate:"required,url"`
}

type GetUrlsQuery struct {
	Page     int `validate:"omitempty,min=1"`
	PageSize int `validate:"omitempty,min=1,max=100"`
}

func (h *UrlHandlerWriter) PostUrl(w http.ResponseWriter, r *http.Request) {
	var req CreateOrUpdateURLRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(
			w,
			errs.ProblemInvalidJSONBody,
			"invalid json body",
			http.StatusBadRequest,
			"request body contains invalid JSON",
		)
		return
	}

	err := h.validate.Struct(req)
	if err != nil {
		h.log.Warn("invalid request body", "error", err)
		writeError(
			w,
			errs.ProblemValidation,
			"validation error",
			http.StatusBadRequest,
			"request body contains invalid data",
		)
		return
	}

	resp, err := h.service.PostUrl(r.Context(), req.Url)
	if err != nil {
		if errors.Is(err, errs.ErrAlreadyExists) {
			h.log.Info("url already exists", "url", req.Url)
			writeError(
				w,
				errs.ProblemConflict,
				"status conflict",
				http.StatusConflict,
				"url already exists",
			)
			return
		}
		h.log.Error("failed to create url", "url", req.Url, "error", err)
		writeError(
			w,
			errs.ProblemInternal,
			"internal server error",
			http.StatusInternalServerError,
			"an internal server error occurred",
		)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.log.Error("failed to encode json", "error", err)
	}
}

func (h *UrlHandlerReader) GetUrl(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")

	err := h.validate.Var(code, "required,len=8,alphanum")
	if err != nil {
		h.log.Warn("invalid code", "code", code, "error", err)
		writeError(
			w,
			errs.ProblemValidation,
			"validation error",
			http.StatusBadRequest,
			"code must contain exactly 8 alphanumeric characters",
		)
		return
	}

	resp, err := h.service.GetUrl(r.Context(), code)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			h.log.Info("url not found", "code", code)
			writeError(
				w,
				errs.ProblemNotFound,
				"not found",
				http.StatusNotFound,
				"URL not found",
			)
			return
		}
		h.log.Error("failed to get url", "code", code, "error", err)
		writeError(
			w,
			errs.ProblemInternal,
			"internal server error",
			http.StatusInternalServerError,
			"an internal server error occurred",
		)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.log.Error("failed to encode json", "error", err)
	}
}

func (h *UrlHandlerReader) GetAllUrlsWithStatistics(w http.ResponseWriter, r *http.Request) {
	filter := model.UrlFilter{
		Sort:    "date",
		OrderBy: "asc",
	}

	if search := r.URL.Query().Get("search"); search != "" {
		filter.Search = &search
	}

	if dateFrom := r.URL.Query().Get("date_from"); dateFrom != "" {
		layout := "2006-01-02"
		df, err := time.Parse(layout, dateFrom)
		if err != nil {
			writeError(
				w,
				errs.ProblemValidation,
				"validation error",
				http.StatusBadRequest,
				"invalid 'date_from' format, use YYYY-MM-DD",
			)
			return
		}
		filter.DateFrom = &df
	}

	if dateTo := r.URL.Query().Get("date_to"); dateTo != "" {
		layout := "2006-01-02"
		dt, err := time.Parse(layout, dateTo)
		if err != nil {
			writeError(
				w,
				errs.ProblemValidation,
				"validation error",
				http.StatusBadRequest,
				"invalid 'date_from' format, use YYYY-MM-DD",
			)
			return
		}
		filter.DateTo = &dt
	}

	if minClicks := r.URL.Query().Get("min_clicks"); minClicks != "" {
		minClicksInt, err := strconv.Atoi(minClicks)
		if err != nil {
			writeError(
				w,
				errs.ProblemValidation,
				"validation error",
				http.StatusBadRequest,
				"invalid 'min_clicks', must be an integer",
			)
			return
		}
		filter.MinClicks = &minClicksInt
	}

	if maxClicks := r.URL.Query().Get("max_clicks"); maxClicks != "" {
		maxClicksInt, err := strconv.Atoi(maxClicks)
		if err != nil {
			writeError(
				w,
				errs.ProblemValidation,
				"validation error",
				http.StatusBadRequest,
				"invalid 'max_clicks' , must be an integer",
			)
			return
		}
		filter.MaxClicks = &maxClicksInt
	}

	if sort := r.URL.Query().Get("sort"); sort != "" {
		filter.Sort = sort
	}

	if orderBy := r.URL.Query().Get("order_by"); orderBy != "" {
		filter.OrderBy = orderBy
	}

	if err := h.validate.Struct(filter); err != nil {
		h.log.Error("failed to validate filter struct", "error", err)
		writeError(
			w,
			errs.ProblemValidation,
			"validation error",
			http.StatusBadRequest,
			"invalid filter parameters",
		)
		return
	}

	queryPagination := GetUrlsQuery{
		Page:     1,
		PageSize: 20,
	}

	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		queryPagination.Page, _ = strconv.Atoi(pageStr)
	}

	if pageSizeStr := r.URL.Query().Get("page_size"); pageSizeStr != "" {
		queryPagination.PageSize, _ = strconv.Atoi(pageSizeStr)
	}

	if err := h.validate.Struct(queryPagination); err != nil {
		h.log.Error("failed to convert query param", "error", err)
		writeError(
			w,
			errs.ProblemValidation,
			"validation error",
			http.StatusBadRequest,
			"invalid pagination parameters. 'page' must be >= 1, 'page_size' must be 1-100",
		)
		return
	}

	resp, err := h.service.GetAllUrlsWithStatistics(r.Context(), queryPagination.Page, queryPagination.PageSize, filter)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			h.log.Info("urls not found")
			writeError(
				w,
				errs.ProblemNotFound,
				"not found",
				http.StatusNotFound,
				"URLs not found",
			)
			return
		}
		h.log.Error("failed to get urls", "error", err)
		writeError(
			w,
			errs.ProblemInternal,
			"internal server error",
			http.StatusInternalServerError,
			"an internal server error occurred",
		)
		return
	}

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.log.Error("failed to encode json", "error", err)
	}
}

func (h *UrlHandlerWriter) PutUrl(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")

	if err := h.validate.Var(code, "required,len=8,alphanum"); err != nil {
		h.log.Warn("invalid short code", "code", code, "error", err)
		writeError(
			w,
			errs.ProblemValidation,
			"validation error",
			http.StatusBadRequest,
			"code must contain exactly 8 alphanumeric characters",
		)
		return
	}

	var req CreateOrUpdateURLRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Warn("invalid json body", "error", err)
		writeError(
			w,
			errs.ProblemInvalidJSONBody,
			"invalid json body",
			http.StatusBadRequest,
			"request body contains invalid JSON",
		)
		return
	}

	err := h.validate.Struct(req)
	if err != nil {
		h.log.Warn("invalid request body", "error", err)
		writeError(
			w,
			errs.ProblemValidation,
			"validation error",
			http.StatusBadRequest,
			"request body contains invalid data",
		)
		return
	}

	resp, err := h.service.PutUrl(r.Context(), code, req.Url)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			h.log.Info("url not found", "code", code)
			writeError(
				w,
				errs.ProblemNotFound,
				"not found",
				http.StatusNotFound,
				"URL not found",
			)
			return
		}
		h.log.Error(
			"failed to update url",
			"code", code,
			"error", err,
		)
		writeError(
			w,
			errs.ProblemInternal,
			"internal server error",
			http.StatusInternalServerError,
			"an internal server error occurred",
		)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.log.Error("failed to encode json", "error", err)
	}
}

func (h *UrlHandlerWriter) DeleteUrl(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")

	if err := h.validate.Var(code, "required,len=8,alphanum"); err != nil {
		h.log.Warn("invalid short code", "code", code, "error", err)
		writeError(
			w,
			errs.ProblemValidation,
			"validation error",
			http.StatusBadRequest,
			"code must contain exactly 8 alphanumeric characters",
		)
		return
	}

	err := h.service.DeleteUrl(r.Context(), code)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			h.log.Info("url not found", "code", code)
			writeError(
				w,
				errs.ProblemNotFound,
				"not found",
				http.StatusNotFound,
				"URL not found",
			)
			return
		}
		h.log.Error(
			"failed to delete url",
			"code", code,
			"error", err,
		)
		writeError(
			w,
			errs.ProblemInternal,
			"internal server error",
			http.StatusInternalServerError,
			"an internal server error occurred",
		)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *UrlHandlerReader) GetUrlStatistics(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")

	if err := h.validate.Var(code, "required,len=8,alphanum"); err != nil {
		writeError(
			w,
			errs.ProblemValidation,
			"validation error",
			http.StatusBadRequest,
			"code must contain exactly 8 alphanumeric characters",
		)
		return
	}

	resp, err := h.service.GetUrlStatistics(r.Context(), code)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			h.log.Info("url not found", "code", code)
			writeError(
				w,
				errs.ProblemNotFound,
				"not found",
				http.StatusNotFound,
				"URL not found",
			)
			return
		}
		writeError(
			w,
			errs.ProblemInternal,
			"internal server error",
			http.StatusInternalServerError,
			"an internal server error occurred",
		)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.log.Error("failed to encode json", "error", err)
	}
}

func (h *UrlHandlerReader) RedirectUrl(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")

	if err := h.validate.Var(code, "required,len=8,alphanum"); err != nil {
		h.log.Warn("invalid short code", "error", err)
		writeError(
			w,
			errs.ProblemValidation,
			"validation error",
			http.StatusBadRequest,
			"code must contain exactly 8 alphanumeric characters",
		)
		return
	}

	resp, err := h.service.GetUrl(r.Context(), code)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			writeError(
				w,
				errs.ProblemNotFound,
				"not found",
				http.StatusNotFound,
				"URL not found",
			)
			return
		}
		writeError(
			w,
			errs.ProblemInternal,
			"internal server error",
			http.StatusInternalServerError,
			"an internal server error occurred",
		)
		return
	}

	http.Redirect(w, r, resp.Url, http.StatusFound)
}

func writeError(w http.ResponseWriter, typeError, title string, status int, detail string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	json.NewEncoder(w).Encode(errs.ErrorResponse{
		Type:   typeError,
		Title:  title,
		Status: status,
		Detail: detail,
	})
}
