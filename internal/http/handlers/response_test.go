package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sunriseex/finance-manager/internal/http/dto"
	"github.com/sunriseex/finance-manager/internal/services"
)

func TestDecodeOptionalJSONAllowsEmptyBody(t *testing.T) {
	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/api/accounts/account-1/accrue-interest",
		http.NoBody,
	)

	var body dto.AccrueInterestRequest
	if err := decodeOptionalJSON(req, &body); err != nil {
		t.Fatalf("decode optional json: %v", err)
	}

	if body.RuleID != "" {
		t.Fatalf("rule id = %q, want empty", body.RuleID)
	}
	if body.Date != "" {
		t.Fatalf("date = %q, want empty", body.Date)
	}
}

func TestDecodeOptionalJSONDecodesValidBody(t *testing.T) {
	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/api/accounts/account-1/accrue-interest",
		strings.NewReader(`{"rule_id":"rule-1","date":"2026-05-06"}`),
	)

	var body dto.AccrueInterestRequest
	if err := decodeOptionalJSON(req, &body); err != nil {
		t.Fatalf("decode optional json: %v", err)
	}

	if body.RuleID != "rule-1" {
		t.Fatalf("rule id = %q, want rule-1", body.RuleID)
	}
	if body.Date != "2026-05-06" {
		t.Fatalf("date = %q, want 2026-05-06", body.Date)
	}
}

func TestDecodeOptionalJSONRejectsMalformedBody(t *testing.T) {
	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/api/accounts/account-1/accrue-interest",
		strings.NewReader(`{"rule_id":`),
	)

	var body dto.AccrueInterestRequest
	if err := decodeOptionalJSON(req, &body); err == nil {
		t.Fatal("expected error")
	}
}

func TestDecodeOptionalJSONRejectsUnknownFields(t *testing.T) {
	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/api/accounts/account-1/accrue-interest",
		strings.NewReader(`{"unknown":true}`),
	)

	var body dto.AccrueInterestRequest
	if err := decodeOptionalJSON(req, &body); err == nil {
		t.Fatal("expected error")
	}
}

func TestDecodeJSONRejectsTrailingData(t *testing.T) {
	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/api/accounts",
		strings.NewReader(`{"name":"A"}{"name":"B"}`),
	)

	var body struct {
		Name string `json:"name"`
	}

	if err := decodeJSON(req, &body); err == nil {
		t.Fatal("expected error")
	}
}

func TestDecodeJSONRejectsUnknownFields(t *testing.T) {
	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/api/accounts",
		strings.NewReader(`{"name":"A","unknown":true}`),
	)

	var body struct {
		Name string `json:"name"`
	}

	if err := decodeJSON(req, &body); err == nil {
		t.Fatal("expected error")
	}
}

func TestWriteValidationOrServiceErrorWritesValidationError(t *testing.T) {
	rec := httptest.NewRecorder()

	writeValidationOrServiceError(rec, services.ValidationError("amount must be non-zero"))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	var body dto.ErrorEnvelope
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if body.Error.Code != "validation_error" {
		t.Fatalf("code = %q, want validation_error", body.Error.Code)
	}
}

func TestWriteValidationOrServiceErrorWritesInternalErrorForRegularError(t *testing.T) {
	rec := httptest.NewRecorder()

	writeValidationOrServiceError(rec, errors.New("database failed"))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}

	var body dto.ErrorEnvelope
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if body.Error.Code != "internal_error" {
		t.Fatalf("code = %q, want internal_error", body.Error.Code)
	}
}
