package models

import "time"

type IdempotencyRecord struct {
	Key          string
	UserID       string
	Method       string
	Path         string
	RequestHash  string
	StatusCode   *int
	ResponseBody []byte
	CreatedAt    time.Time
	ExpiresAt    time.Time
}
