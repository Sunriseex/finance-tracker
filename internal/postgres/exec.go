package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type sqlExecer interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

type queryExecer interface {
	sqlExecer
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}
