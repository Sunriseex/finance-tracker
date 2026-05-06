package dto

import (
	"encoding/json"
	"testing"
)

func TestNullableInt64DistinguishesOmittedNullAndValue(t *testing.T) {
	var omitted struct {
		PromoRate NullableInt64 `json:"promo_rate_bps"`
	}
	if err := json.Unmarshal([]byte(`{}`), &omitted); err != nil {
		t.Fatalf("decode omitted: %v", err)
	}
	if omitted.PromoRate.Set {
		t.Fatalf("omitted field = %+v, want Set=false", omitted.PromoRate)
	}

	var nullValue struct {
		PromoRate NullableInt64 `json:"promo_rate_bps"`
	}
	if err := json.Unmarshal([]byte(`{"promo_rate_bps":null}`), &nullValue); err != nil {
		t.Fatalf("decode null: %v", err)
	}
	if !nullValue.PromoRate.Set || nullValue.PromoRate.Valid {
		t.Fatalf("null field = %+v, want Set=true Valid=false", nullValue.PromoRate)
	}

	var realValue struct {
		PromoRate NullableInt64 `json:"promo_rate_bps"`
	}
	if err := json.Unmarshal([]byte(`{"promo_rate_bps":1500}`), &realValue); err != nil {
		t.Fatalf("decode value: %v", err)
	}
	if !realValue.PromoRate.Set || !realValue.PromoRate.Valid || realValue.PromoRate.Value != 1500 {
		t.Fatalf("value field = %+v, want Set=true Valid=true Value=1500", realValue.PromoRate)
	}
}

func TestNullableStringDistinguishesOmittedNullAndValue(t *testing.T) {
	var omitted struct {
		EndDate NullableString `json:"end_date"`
	}
	if err := json.Unmarshal([]byte(`{}`), &omitted); err != nil {
		t.Fatalf("decode omitted: %v", err)
	}
	if omitted.EndDate.Set {
		t.Fatalf("omitted field = %+v, want Set=false", omitted.EndDate)
	}

	var nullValue struct {
		EndDate NullableString `json:"end_date"`
	}
	if err := json.Unmarshal([]byte(`{"end_date":null}`), &nullValue); err != nil {
		t.Fatalf("decode null: %v", err)
	}
	if !nullValue.EndDate.Set || nullValue.EndDate.Valid {
		t.Fatalf("null field = %+v, want Set=true Valid=false", nullValue.EndDate)
	}

	var realValue struct {
		EndDate NullableString `json:"end_date"`
	}
	if err := json.Unmarshal([]byte(`{"end_date":"2026-06-01"}`), &realValue); err != nil {
		t.Fatalf("decode value: %v", err)
	}
	if !realValue.EndDate.Set || !realValue.EndDate.Valid || realValue.EndDate.Value != "2026-06-01" {
		t.Fatalf("value field = %+v, want Set=true Valid=true Value=2026-06-01", realValue.EndDate)
	}
}
