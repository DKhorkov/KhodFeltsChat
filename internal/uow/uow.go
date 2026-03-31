package uow

import (
	"context"
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
	fn func(ctx context.Context, tx pg.Transaction) error,
) error {
	tx, err := uow.pg.Transaction(ctx, uow.opts...)
	if err != nil {
		return err
	}

	// Буферизованный канал, чтобы пишущая горутина не блокировлась и сразу завершилась.
	// Иначе может быть кейс, что основная функция отработала по контексту.
	// Далее выход из функции, а канал не закрыт, горутина заблокирована. Происходит утечка.
	errChan := make(chan error, 1)

	go func() {
		defer close(errChan)

		errChan <- fn(ctx, tx)
	}()

	select {
	case <-ctx.Done():
		err = tx.Rollback()
		if err != nil {
			return fmt.Errorf("%w: %w", ctx.Err(), err)
		}

		return ctx.Err()
	case err = <-errChan:
		if err != nil {
			rollbackErr := tx.Rollback()
			if rollbackErr != nil {
				err = fmt.Errorf("%w: %w", err, rollbackErr)
			}

			return err
		}

		return tx.Commit()
	}
}
