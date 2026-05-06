package dto

import (
	"bytes"
	"encoding/json"
	"fmt"
)

type NullableInt64 struct {
	Set   bool
	Valid bool
	Value int64
}

func (n *NullableInt64) UnmarshalJSON(data []byte) error {
	n.Set = true

	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		n.Valid = false
		n.Value = 0
		return nil
	}

	if err := json.Unmarshal(data, &n.Value); err != nil {
		return fmt.Errorf("decode nullable int64: %w", err)
	}

	n.Valid = true
	return nil
}

type NullableString struct {
	Set   bool
	Valid bool
	Value string
}

func (n *NullableString) UnmarshalJSON(data []byte) error {
	n.Set = true

	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		n.Valid = false
		n.Value = ""
		return nil
	}

	if err := json.Unmarshal(data, &n.Value); err != nil {
		return fmt.Errorf("decode nullable string: %w", err)
	}

	n.Valid = true
	return nil
}
