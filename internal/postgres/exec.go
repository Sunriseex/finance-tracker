package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgconn"
)

type sqlExecer interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}
