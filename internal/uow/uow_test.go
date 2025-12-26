package uow_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DKhorkov/kfc/internal/uow"
	"github.com/DKhorkov/libs/db/postgresql"
	mockpostgresql "github.com/DKhorkov/libs/db/postgresql/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestUnitOfWork_Do(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		fn         func(ctx context.Context, tx postgresql.Transaction) error
		ctx        context.Context
		setupMocks func(conn *mockpostgresql.MockConnector, tx *mockpostgresql.MockTransaction)
		wantErr    bool
		errCheck   func(t *testing.T, err error)
	}{
		{
			name: "успешное выполнение транзакции",
			fn: func(_ context.Context, _ postgresql.Transaction) error {
				return nil
			},
			ctx: context.Background(),
			setupMocks: func(conn *mockpostgresql.MockConnector, tx *mockpostgresql.MockTransaction) {
				tx.EXPECT().Commit().Return(nil)
				conn.EXPECT().Transaction(gomock.Any(), gomock.Any()).Return(tx, nil)
			},
			wantErr: false,
		},
		{
			name: "ошибка при создании транзакции",
			fn:   nil,
			ctx:  context.Background(),
			setupMocks: func(conn *mockpostgresql.MockConnector, _ *mockpostgresql.MockTransaction) {
				conn.EXPECT().Transaction(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("connection failed"))
			},
			wantErr: true,
			errCheck: func(t *testing.T, err error) {
				t.Helper()
				assert.Contains(t, err.Error(), "connection failed")
			},
		},
		{
			name: "ошибка в бизнес-логике",
			fn: func(_ context.Context, _ postgresql.Transaction) error {
				return errors.New("business logic failed")
			},
			ctx: context.Background(),
			setupMocks: func(conn *mockpostgresql.MockConnector, tx *mockpostgresql.MockTransaction) {
				tx.EXPECT().Rollback().Return(nil)
				conn.EXPECT().Transaction(gomock.Any(), gomock.Any()).Return(tx, nil)
			},
			wantErr: true,
			errCheck: func(t *testing.T, err error) {
				t.Helper()
				assert.Contains(t, err.Error(), "business logic failed")
			},
		},
		{
			name: "ошибка отката транзакции при ошибкe в бизнес-логике",
			fn: func(_ context.Context, _ postgresql.Transaction) error {
				return errors.New("business logic failed")
			},
			ctx: context.Background(),
			setupMocks: func(conn *mockpostgresql.MockConnector, tx *mockpostgresql.MockTransaction) {
				tx.EXPECT().Rollback().Return(errors.New("rollback failed"))
				conn.EXPECT().Transaction(gomock.Any(), gomock.Any()).Return(tx, nil)
			},
			wantErr: true,
			errCheck: func(t *testing.T, err error) {
				t.Helper()
				assert.Contains(t, err.Error(), "business logic failed")
			},
		},
		{
			name: "отмена контекста во время выполнения",
			fn: func(ctx context.Context, _ postgresql.Transaction) error {
				select {
				case <-time.After(200 * time.Millisecond):
					return nil
				case <-ctx.Done():
					return ctx.Err()
				}
			},
			ctx: func() context.Context {
				ctx, _ := context.WithDeadline(
					context.Background(),
					time.Now().Add(time.Millisecond),
				)

				return ctx
			}(),
			setupMocks: func(conn *mockpostgresql.MockConnector, tx *mockpostgresql.MockTransaction) {
				tx.EXPECT().Rollback().Return(nil)
				conn.EXPECT().Transaction(gomock.Any(), gomock.Any()).Return(tx, nil)
			},
			wantErr: true,
			errCheck: func(t *testing.T, err error) {
				assert.ErrorIs(t, err, context.DeadlineExceeded)
			},
		},
		{
			name: "контекст отменен до начала выполнения",
			fn: func(_ context.Context, _ postgresql.Transaction) error {
				time.Sleep(50 * time.Millisecond)

				return nil
			},
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel() // Немедленная отмена

				return ctx
			}(),
			setupMocks: func(conn *mockpostgresql.MockConnector, tx *mockpostgresql.MockTransaction) {
				tx.EXPECT().Rollback().Return(nil)
				conn.EXPECT().Transaction(gomock.Any(), gomock.Any()).Return(tx, nil)
			},
			wantErr: true,
			errCheck: func(t *testing.T, err error) {
				assert.ErrorIs(t, err, context.Canceled)
			},
		},
		{
			name: "ошибка отката транзакции при отмене контекста",
			fn: func(_ context.Context, _ postgresql.Transaction) error {
				time.Sleep(50 * time.Millisecond)

				return nil
			},
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel() // Немедленная отмена

				return ctx
			}(),
			setupMocks: func(conn *mockpostgresql.MockConnector, tx *mockpostgresql.MockTransaction) {
				tx.EXPECT().Rollback().Return(errors.New("rollback failed"))
				conn.EXPECT().Transaction(gomock.Any(), gomock.Any()).Return(tx, nil)
			},
			wantErr: true,
			errCheck: func(t *testing.T, err error) {
				assert.ErrorIs(t, err, context.Canceled)
			},
		},
		{
			name: "быстрое завершение бизнес-логики до отмены контекста",
			fn: func(_ context.Context, _ postgresql.Transaction) error {
				// Быстрая операция
				return nil
			},
			ctx: func() context.Context {
				ctx, _ := context.WithTimeout(context.Background(), 100*time.Millisecond)

				return ctx
			}(),
			setupMocks: func(conn *mockpostgresql.MockConnector, tx *mockpostgresql.MockTransaction) {
				tx.EXPECT().Commit().Return(nil)
				conn.EXPECT().Transaction(gomock.Any(), gomock.Any()).Return(tx, nil)
			},
			wantErr: false,
		},
		{
			name: "бизнес-логика возвращает nil, но commit возвращает ошибку",
			fn: func(_ context.Context, _ postgresql.Transaction) error {
				return nil
			},
			ctx: context.Background(),
			setupMocks: func(conn *mockpostgresql.MockConnector, tx *mockpostgresql.MockTransaction) {
				tx.EXPECT().Commit().Return(errors.New("commit failed"))
				conn.EXPECT().Transaction(gomock.Any(), gomock.Any()).Return(tx, nil)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			conn := mockpostgresql.NewMockConnector(ctrl)
			tx := mockpostgresql.NewMockTransaction(ctrl)

			if tt.setupMocks != nil {
				tt.setupMocks(conn, tx)
			}

			unitOfWork := uow.New(conn, nil)

			err := unitOfWork.Do(tt.ctx, tt.fn)
			if tt.wantErr {
				require.Error(t, err)

				if tt.errCheck != nil {
					tt.errCheck(t, err)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}
