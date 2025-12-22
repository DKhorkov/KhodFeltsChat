package interfaces

import (
	"context"
	"database/sql"
)

type UnitOfWork interface {
	Do(ctx context.Context, fn func(ctx context.Context, tx *sql.Tx) error) error
}
