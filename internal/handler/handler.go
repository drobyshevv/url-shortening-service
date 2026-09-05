package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/drobyshevv/url-shortening-service/internal/model"
	"github.com/go-playground/validator/v10"
)

type UrlServiceWriter interface {
	PostUrl(context.Context, string) (*model.Url, error)
	PutUrl(context.Context, string, string) (*model.Url, error)
	DeleteUrl(context.Context, string) error
}

type UrlServiceReader interface {
	GetUrl(context.Context, string) (*model.Url, error)
	GetUrlStatistics(context.Context, string) (*model.UrlStats, error)
	GetAllUrlsWithStatistics(context.Context, string) ([]model.UrlStats, error)
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

func (h *UrlHandlerWriter) PostUrl(w http.ResponseWriter, r *http.Request) {
	var req CreateOrUpdateURLRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(
			w,
			model.ProblemInvalidJSONBody,
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
			model.ProblemValidation,
			"validation error",
			http.StatusBadRequest,
			"request body contains invalid data",
		)
		return
	}

	resp, err := h.service.PostUrl(r.Context(), req.Url)
	if err != nil {
		if errors.Is(err, model.ErrAlreadyExists) {
			h.log.Info("url already exists", "url", req.Url)
			writeError(
				w,
				model.ProblemConflict,
				"status conflict",
				http.StatusConflict,
				"url already exists",
			)
			return
		}
		h.log.Error("failed to create url", "url", req.Url, "error", err)
		writeError(
			w,
			model.ProblemInternal,
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
			model.ProblemValidation,
			"validation error",
			http.StatusBadRequest,
			"code must contain exactly 8 alphanumeric characters",
		)
		return
	}

	resp, err := h.service.GetUrl(r.Context(), code)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			h.log.Info("url not found", "code", code)
			writeError(
				w,
				model.ProblemNotFound,
				"not found",
				http.StatusNotFound,
				"URL not found",
			)
			return
		}
		h.log.Error("failed to get url", "code", code, "error", err)
		writeError(
			w,
			model.ProblemInternal,
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
	//get all urls

	//validation structs
	//decode
	//response
}

func (h *UrlHandlerWriter) PutUrl(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")

	if err := h.validate.Var(code, "required,len=8,alphanum"); err != nil {
		h.log.Warn("invalid short code", "code", code, "error", err)
		writeError(
			w,
			model.ProblemValidation,
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
			model.ProblemInvalidJSONBody,
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
			model.ProblemValidation,
			"validation error",
			http.StatusBadRequest,
			"request body contains invalid data",
		)
		return
	}

	resp, err := h.service.PutUrl(r.Context(), code, req.Url)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			h.log.Info("url not found", "code", code)
			writeError(
				w,
				model.ProblemNotFound,
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
			model.ProblemInternal,
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
			model.ProblemValidation,
			"validation error",
			http.StatusBadRequest,
			"code must contain exactly 8 alphanumeric characters",
		)
		return
	}

	err := h.service.DeleteUrl(r.Context(), code)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			h.log.Info("url not found", "code", code)
			writeError(
				w,
				model.ProblemNotFound,
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
			model.ProblemInternal,
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
			model.ProblemValidation,
			"validation error",
			http.StatusBadRequest,
			"code must contain exactly 8 alphanumeric characters",
		)
		return
	}

	resp, err := h.service.GetUrlStatistics(r.Context(), code)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			h.log.Info("url not found", "code", code)
			writeError(
				w,
				model.ProblemNotFound,
				"not found",
				http.StatusNotFound,
				"URL not found",
			)
			return
		}
		writeError(
			w,
			model.ProblemInternal,
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
			model.ProblemValidation,
			"validation error",
			http.StatusBadRequest,
			"code must contain exactly 8 alphanumeric characters",
		)
		return
	}

	resp, err := h.service.GetUrl(r.Context(), code)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			writeError(
				w,
				model.ProblemNotFound,
				"not found",
				http.StatusNotFound,
				"URL not found",
			)
			return
		}
		writeError(
			w,
			model.ProblemInternal,
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

	json.NewEncoder(w).Encode(model.ErrorResponse{
		Type:   typeError,
		Title:  title,
		Status: status,
		Detail: detail,
	})
}
