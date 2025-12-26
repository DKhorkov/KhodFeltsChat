package interfaces

import (
	"context"

	pg "github.com/DKhorkov/libs/db/postgresql"
)

//go:generate mockgen -source=uow.go -destination=../../mocks/uow/uow.go -package=mockunitofwork -exclude_interfaces=
type UnitOfWork interface {
	Do(ctx context.Context, fn func(ctx context.Context, tx pg.Transaction) error) error
}
