package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sunriseex/capitalflow/internal/auth"
	"github.com/sunriseex/capitalflow/internal/models"
	"github.com/sunriseex/capitalflow/internal/repository"
)

func TestIdempotencyFlushesResponseAfterComplete(t *testing.T) {
	repo := newTestIdempotencyRepo()
	handler := Idempotency(repo)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	req := newIdempotencyRequest(t, "create-transaction")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("content type = %q, want application/json", got)
	}
	if got := rec.Body.String(); got != `{"ok":true}` {
		t.Fatalf("body = %q", got)
	}
	if repo.completeCalls != 1 {
		t.Fatalf("complete calls = %d, want 1", repo.completeCalls)
	}
}

func TestIdempotencyReturnsCompletionUnknownWhenCompleteFailsAfterSuccess(t *testing.T) {
	repo := newTestIdempotencyRepo()
	repo.completeErr = errors.New("database unavailable")
	handler := Idempotency(repo)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"created":true}`))
	}))
	req := newIdempotencyRequest(t, "create-transaction")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusConflict)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("content type = %q, want application/json", got)
	}
	if strings.Contains(rec.Body.String(), "created") {
		t.Fatalf("handler response was flushed after failed Complete: %s", rec.Body.String())
	}
	var body struct {
		Error struct {
			Code    string         `json:"code"`
			Message string         `json:"message"`
			Details map[string]any `json:"details"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Error.Code != "idempotency_completion_unknown" {
		t.Fatalf("code = %q, want idempotency_completion_unknown", body.Error.Code)
	}
	if body.Error.Message != idempotencyCompletionUnknownMessage {
		t.Fatalf("message = %q", body.Error.Message)
	}
	if len(body.Error.Details) != 0 {
		t.Fatalf("details = %#v, want empty", body.Error.Details)
	}
	record := repo.records["create-transaction\x00user-1\x00POST\x00/api/v1/transactions"]
	if record == nil {
		t.Fatal("idempotency record was not created")
	}
	if record.StatusCode != nil {
		t.Fatalf("status code was stored despite Complete failure: %d", *record.StatusCode)
	}
}

type testIdempotencyRepo struct {
	records       map[string]*models.IdempotencyRecord
	completeErr   error
	completeCalls int
}

func newTestIdempotencyRepo() *testIdempotencyRepo {
	return &testIdempotencyRepo{records: map[string]*models.IdempotencyRecord{}}
}

func (r *testIdempotencyRepo) Get(_ context.Context, key, userID, method, path string) (*models.IdempotencyRecord, error) {
	record, ok := r.records[idempotencyTestKey(key, userID, method, path)]
	if !ok {
		return nil, repository.ErrNotFound
	}
	recordCopy := *record
	recordCopy.ResponseBody = append([]byte(nil), record.ResponseBody...)
	return &recordCopy, nil
}

func (r *testIdempotencyRepo) CreatePending(_ context.Context, record *models.IdempotencyRecord) (bool, error) {
	key := idempotencyTestKey(record.Key, record.UserID, record.Method, record.Path)
	if _, ok := r.records[key]; ok {
		return false, nil
	}
	recordCopy := *record
	r.records[key] = &recordCopy
	return true, nil
}

func (r *testIdempotencyRepo) Complete(_ context.Context, key, userID, method, path string, statusCode int, responseBody []byte) error {
	r.completeCalls++
	if r.completeErr != nil {
		return r.completeErr
	}
	record, ok := r.records[idempotencyTestKey(key, userID, method, path)]
	if !ok {
		return repository.ErrNotFound
	}
	record.StatusCode = &statusCode
	record.ResponseBody = append([]byte(nil), responseBody...)
	return nil
}

func newIdempotencyRequest(t *testing.T, key string) *http.Request {
	t.Helper()

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/transactions", strings.NewReader(`{"amount_minor":100}`))
	req.Header.Set(IdempotencyKeyHeader, key)
	ctx := context.WithValue(req.Context(), userClaimsKey, &auth.Claims{
		UserID:    "user-1",
		Email:     "user@example.com",
		SessionID: "session-1",
		TokenType: auth.TokenTypeAccess,
	})
	return req.WithContext(ctx)
}

func idempotencyTestKey(key, userID, method, path string) string {
	return key + "\x00" + userID + "\x00" + method + "\x00" + path
}
