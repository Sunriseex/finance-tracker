package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sunriseex/finance-manager/internal/http/dto"
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
