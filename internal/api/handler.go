package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/kasbench/globeco-fix-engine/internal/domain"
	"github.com/kasbench/globeco-fix-engine/internal/repository"
)

type ExecutionAPI struct {
	Repo repository.ExecutionRepository
}

func NewExecutionAPI(repo repository.ExecutionRepository) *ExecutionAPI {
	return &ExecutionAPI{Repo: repo}
}

func (h *ExecutionAPI) ListExecutions(w http.ResponseWriter, r *http.Request) {
	execs, err := h.Repo.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list executions")
		return
	}
	var dtos []*domain.ExecutionDTO
	for _, exec := range execs {
		dtos = append(dtos, domain.MapExecutionToDTO(exec))
	}
	writeJSON(w, http.StatusOK, dtos)
}

func (h *ExecutionAPI) GetExecutionByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	exec, err := h.Repo.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "execution not found")
		return
	}
	dto := domain.MapExecutionToDTO(exec)
	writeJSON(w, http.StatusOK, dto)
}

func (h *ExecutionAPI) RegisterRoutes(r chi.Router) {
	r.Get("/api/v1/executions", h.ListExecutions)
	r.Route("/api/v1/execution", func(r chi.Router) {
		r.Get("/{id}", h.GetExecutionByID)
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	data, err := json.Marshal(v)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"failed to encode response"}`))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
