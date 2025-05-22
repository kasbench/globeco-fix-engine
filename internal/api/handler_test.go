package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/kasbench/globeco-fix-engine/internal/domain"
	"github.com/kasbench/globeco-fix-engine/internal/repository"
	"github.com/stretchr/testify/assert"
)

type mockRepo struct {
	execs []*repository.Execution
}

func (m *mockRepo) Create(ctx context.Context, exec *repository.Execution) error { return nil }
func (m *mockRepo) GetByID(ctx context.Context, id int) (*repository.Execution, error) {
	for _, e := range m.execs {
		if e.ID == id {
			return e, nil
		}
	}
	return nil, http.ErrNoLocation
}
func (m *mockRepo) List(ctx context.Context) ([]*repository.Execution, error) { return m.execs, nil }
func (m *mockRepo) PollNextForFill(ctx context.Context) (*repository.Execution, error) {
	return nil, nil
}
func (m *mockRepo) Update(ctx context.Context, exec *repository.Execution) error { return nil }

func TestListExecutions(t *testing.T) {
	repo := &mockRepo{
		execs: []*repository.Execution{
			{ID: 1, Ticker: "AAPL", ExecutionStatus: "WORK"},
			{ID: 2, Ticker: "GOOG", ExecutionStatus: "FULL"},
		},
	}
	h := NewExecutionAPI(repo)
	r := chi.NewRouter()
	h.RegisterRoutes(r)

	req := httptest.NewRequest("GET", "/api/v1/executions", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var dtos []domain.ExecutionDTO
	err := json.NewDecoder(w.Body).Decode(&dtos)
	assert.NoError(t, err)
	assert.Len(t, dtos, 2)
	assert.Equal(t, "AAPL", dtos[0].Ticker)
	assert.Equal(t, "GOOG", dtos[1].Ticker)
}

func TestGetExecutionByID(t *testing.T) {
	repo := &mockRepo{
		execs: []*repository.Execution{
			{ID: 1, Ticker: "AAPL", ExecutionStatus: "WORK"},
		},
	}
	h := NewExecutionAPI(repo)
	r := chi.NewRouter()
	h.RegisterRoutes(r)

	req := httptest.NewRequest("GET", "/api/v1/execution/1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var dto domain.ExecutionDTO
	err := json.NewDecoder(w.Body).Decode(&dto)
	assert.NoError(t, err)
	assert.Equal(t, "AAPL", dto.Ticker)

	// Test not found
	req = httptest.NewRequest("GET", "/api/v1/execution/999", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.True(t, strings.Contains(w.Body.String(), "execution not found"))
}
