package uow

import (
	"context"
	"database/sql"
	"fmt"

	pg "github.com/DKhorkov/libs/db/postgresql"
)

type UnitOfWork struct {
	pg   pg.Connector
	opts []pg.TransactionOption
}

func New(pg pg.Connector, opts ...pg.TransactionOption) *UnitOfWork {
	return &UnitOfWork{
		pg:   pg,
		opts: opts,
	}
}

func (uow *UnitOfWork) Do(
	ctx context.Context,
	f func(ctx context.Context, tx *sql.Tx) error,
) error {
	tx, err := uow.pg.Transaction(ctx, uow.opts...)
	if err != nil {
		return err
	}

	doneChan := make(chan struct{})
	errChan := make(chan error)
	go func() {
		defer close(doneChan)
		defer close(errChan)

		err = f(ctx, tx)
		if err != nil {
			errChan <- err
		}

		doneChan <- struct{}{}
	}()

	select {
	case <-doneChan:
		return tx.Commit()
	case <-ctx.Done():
		if closeErr := tx.Rollback(); closeErr != nil {
			return fmt.Errorf("%w: %v", err, closeErr)
		}

		return ctx.Err()
	case err = <-errChan:
		if closeErr := tx.Rollback(); closeErr != nil {
			return fmt.Errorf("%w: %v", err, closeErr)
		}

		return err
	}
}
