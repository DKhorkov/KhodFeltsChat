package push_subscriptions_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DKhorkov/kfc/internal/domains"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/kfc/internal/interfaces"
	service "github.com/DKhorkov/kfc/internal/services/push_subscriptions"
	mockrepositories "github.com/DKhorkov/kfc/mocks/repositories"
	mockuow "github.com/DKhorkov/kfc/mocks/uow"
	pg "github.com/DKhorkov/libs/db/postgresql"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestService_CreatePushSubscription(t *testing.T) {
	t.Parallel()

	type fields struct {
		mockUOW                          func(*mockuow.MockUnitOfWork)
		mockPushSubscriptionsRepository func(*mockrepositories.MockPushSubscriptionsRepository)
	}

	type args struct {
		ctx          context.Context
		subscription domains.PushSubscription
	}

	now := time.Now()

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *domains.PushSubscription
		wantErr bool
		err     error
	}{
		{
			name: "successfully create push subscription",
			fields: fields{
				mockUOW: func(uow *mockuow.MockUnitOfWork) {
					uow.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context, pg.Transaction) error) error {
							tx := &struct{ pg.Transaction }{}

							return fn(ctx, tx)
						})
				},
				mockPushSubscriptionsRepository: func(r *mockrepositories.MockPushSubscriptionsRepository) {
					r.EXPECT().
						CreatePushSubscription(gomock.Any(), domains.PushSubscription{
							UserID:   10,
							Endpoint: "https://example.com/push",
							EncryptionKey:   "key1",
							Auth:     "auth1",
						}).
						Return(uint64(1), nil)

					r.EXPECT().
						GetPushSubscriptionsByUserID(gomock.Any(), uint64(10)).
						Return([]domains.PushSubscription{
							{
								ID:        1,
								UserID:    10,
								Endpoint:  "https://example.com/push",
								EncryptionKey:    "key1",
								Auth:      "auth1",
								CreatedAt: now,
							},
						}, nil)
				},
			},
			args: args{
				ctx: context.Background(),
				subscription: domains.PushSubscription{
					UserID:   10,
					Endpoint: "https://example.com/push",
					EncryptionKey:   "key1",
					Auth:     "auth1",
				},
			},
			want: &domains.PushSubscription{
				ID:        1,
				UserID:    10,
				Endpoint:  "https://example.com/push",
				EncryptionKey:    "key1",
				Auth:      "auth1",
				CreatedAt: now,
			},
			wantErr: false,
		},
		{
			name: "error creating push subscription",
			fields: fields{
				mockUOW: func(uow *mockuow.MockUnitOfWork) {
					uow.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context, pg.Transaction) error) error {
							tx := &struct{ pg.Transaction }{}

							return fn(ctx, tx)
						})
				},
				mockPushSubscriptionsRepository: func(r *mockrepositories.MockPushSubscriptionsRepository) {
					r.EXPECT().
						CreatePushSubscription(gomock.Any(), gomock.Any()).
						Return(uint64(0), errors.New("database error"))
				},
			},
			args: args{
				ctx: context.Background(),
				subscription: domains.PushSubscription{
					UserID:   10,
					Endpoint: "https://example.com/push",
					EncryptionKey:   "key1",
					Auth:     "auth1",
				},
			},
			want:    nil,
			wantErr: true,
			err:     errors.New("database error"),
		},
		{
			name: "error getting subscriptions after create",
			fields: fields{
				mockUOW: func(uow *mockuow.MockUnitOfWork) {
					uow.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context, pg.Transaction) error) error {
							tx := &struct{ pg.Transaction }{}

							return fn(ctx, tx)
						})
				},
				mockPushSubscriptionsRepository: func(r *mockrepositories.MockPushSubscriptionsRepository) {
					r.EXPECT().
						CreatePushSubscription(gomock.Any(), gomock.Any()).
						Return(uint64(1), nil)

					r.EXPECT().
						GetPushSubscriptionsByUserID(gomock.Any(), uint64(10)).
						Return(nil, errors.New("not found"))
				},
			},
			args: args{
				ctx: context.Background(),
				subscription: domains.PushSubscription{
					UserID:   10,
					Endpoint: "https://example.com/push",
					EncryptionKey:   "key1",
					Auth:     "auth1",
				},
			},
			want:    nil,
			wantErr: true,
			err:     errors.New("not found"),
		},
		{
			name: "subscription not found after create",
			fields: fields{
				mockUOW: func(uow *mockuow.MockUnitOfWork) {
					uow.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context, pg.Transaction) error) error {
							tx := &struct{ pg.Transaction }{}

							return fn(ctx, tx)
						})
				},
				mockPushSubscriptionsRepository: func(r *mockrepositories.MockPushSubscriptionsRepository) {
					r.EXPECT().
						CreatePushSubscription(gomock.Any(), gomock.Any()).
						Return(uint64(99), nil)

					r.EXPECT().
						GetPushSubscriptionsByUserID(gomock.Any(), uint64(10)).
						Return([]domains.PushSubscription{
							{
								ID:        1,
								UserID:    10,
								Endpoint:  "https://example.com/push",
								EncryptionKey:    "key1",
								Auth:      "auth1",
								CreatedAt: now,
							},
						}, nil)
				},
			},
			args: args{
				ctx: context.Background(),
				subscription: domains.PushSubscription{
					UserID:   10,
					Endpoint: "https://example.com/push",
					EncryptionKey:   "key1",
					Auth:     "auth1",
				},
			},
			want:    nil,
			wantErr: true,
			err:     customerrors.ErrPushSubscriptionNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			ctrl := gomock.NewController(t)

			mockUOW := mockuow.NewMockUnitOfWork(ctrl)
			mockRepo := mockrepositories.NewMockPushSubscriptionsRepository(ctrl)

			if tt.fields.mockUOW != nil {
				tt.fields.mockUOW(mockUOW)
			}

			if tt.fields.mockPushSubscriptionsRepository != nil {
				tt.fields.mockPushSubscriptionsRepository(mockRepo)
			}

			newRepoFunc := func(_ pg.Transaction) interfaces.PushSubscriptionsRepository {
				return mockRepo
			}

			s := service.New(mockUOW, newRepoFunc)

			// Act
			got, err := s.CreatePushSubscription(tt.args.ctx, tt.args.subscription)

			// Assert
			if tt.wantErr {
				assert.Error(t, err)

				if tt.err != nil {
					assert.Contains(t, err.Error(), tt.err.Error())
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestService_GetPushSubscriptionsByUserID(t *testing.T) {
	t.Parallel()

	type fields struct {
		mockUOW                          func(*mockuow.MockUnitOfWork)
		mockPushSubscriptionsRepository func(*mockrepositories.MockPushSubscriptionsRepository)
	}

	type args struct {
		ctx    context.Context
		userID uint64
	}

	now := time.Now()

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []domains.PushSubscription
		wantErr bool
		err     error
	}{
		{
			name: "successfully get push subscriptions",
			fields: fields{
				mockUOW: func(uow *mockuow.MockUnitOfWork) {
					uow.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context, pg.Transaction) error) error {
							tx := &struct{ pg.Transaction }{}

							return fn(ctx, tx)
						})
				},
				mockPushSubscriptionsRepository: func(r *mockrepositories.MockPushSubscriptionsRepository) {
					r.EXPECT().
						GetPushSubscriptionsByUserID(gomock.Any(), uint64(10)).
						Return([]domains.PushSubscription{
							{
								ID:        1,
								UserID:    10,
								Endpoint:  "https://example.com/push",
								EncryptionKey:    "key1",
								Auth:      "auth1",
								CreatedAt: now,
							},
						}, nil)
				},
			},
			args: args{
				ctx:    context.Background(),
				userID: 10,
			},
			want: []domains.PushSubscription{
				{
					ID:        1,
					UserID:    10,
					Endpoint:  "https://example.com/push",
					EncryptionKey:    "key1",
					Auth:      "auth1",
					CreatedAt: now,
				},
			},
			wantErr: false,
		},
		{
			name: "error getting push subscriptions",
			fields: fields{
				mockUOW: func(uow *mockuow.MockUnitOfWork) {
					uow.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context, pg.Transaction) error) error {
							tx := &struct{ pg.Transaction }{}

							return fn(ctx, tx)
						})
				},
				mockPushSubscriptionsRepository: func(r *mockrepositories.MockPushSubscriptionsRepository) {
					r.EXPECT().
						GetPushSubscriptionsByUserID(gomock.Any(), uint64(10)).
						Return(nil, errors.New("database error"))
				},
			},
			args: args{
				ctx:    context.Background(),
				userID: 10,
			},
			want:    nil,
			wantErr: true,
			err:     errors.New("database error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			ctrl := gomock.NewController(t)

			mockUOW := mockuow.NewMockUnitOfWork(ctrl)
			mockRepo := mockrepositories.NewMockPushSubscriptionsRepository(ctrl)

			if tt.fields.mockUOW != nil {
				tt.fields.mockUOW(mockUOW)
			}

			if tt.fields.mockPushSubscriptionsRepository != nil {
				tt.fields.mockPushSubscriptionsRepository(mockRepo)
			}

			newRepoFunc := func(_ pg.Transaction) interfaces.PushSubscriptionsRepository {
				return mockRepo
			}

			s := service.New(mockUOW, newRepoFunc)

			// Act
			got, err := s.GetPushSubscriptionsByUserID(tt.args.ctx, tt.args.userID)

			// Assert
			if tt.wantErr {
				assert.Error(t, err)

				if tt.err != nil {
					assert.Contains(t, err.Error(), tt.err.Error())
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestService_DeletePushSubscription(t *testing.T) {
	t.Parallel()

	type fields struct {
		mockUOW                          func(*mockuow.MockUnitOfWork)
		mockPushSubscriptionsRepository func(*mockrepositories.MockPushSubscriptionsRepository)
	}

	type args struct {
		ctx context.Context
		id  uint64
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		err     error
	}{
		{
			name: "successfully delete push subscription",
			fields: fields{
				mockUOW: func(uow *mockuow.MockUnitOfWork) {
					uow.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context, pg.Transaction) error) error {
							tx := &struct{ pg.Transaction }{}

							return fn(ctx, tx)
						})
				},
				mockPushSubscriptionsRepository: func(r *mockrepositories.MockPushSubscriptionsRepository) {
					r.EXPECT().
						DeletePushSubscription(gomock.Any(), uint64(1)).
						Return(nil)
				},
			},
			args: args{
				ctx: context.Background(),
				id:  1,
			},
			wantErr: false,
		},
		{
			name: "error deleting push subscription",
			fields: fields{
				mockUOW: func(uow *mockuow.MockUnitOfWork) {
					uow.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context, pg.Transaction) error) error {
							tx := &struct{ pg.Transaction }{}

							return fn(ctx, tx)
						})
				},
				mockPushSubscriptionsRepository: func(r *mockrepositories.MockPushSubscriptionsRepository) {
					r.EXPECT().
						DeletePushSubscription(gomock.Any(), uint64(1)).
						Return(errors.New("database error"))
				},
			},
			args: args{
				ctx: context.Background(),
				id:  1,
			},
			wantErr: true,
			err:     errors.New("database error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			ctrl := gomock.NewController(t)

			mockUOW := mockuow.NewMockUnitOfWork(ctrl)
			mockRepo := mockrepositories.NewMockPushSubscriptionsRepository(ctrl)

			if tt.fields.mockUOW != nil {
				tt.fields.mockUOW(mockUOW)
			}

			if tt.fields.mockPushSubscriptionsRepository != nil {
				tt.fields.mockPushSubscriptionsRepository(mockRepo)
			}

			newRepoFunc := func(_ pg.Transaction) interfaces.PushSubscriptionsRepository {
				return mockRepo
			}

			s := service.New(mockUOW, newRepoFunc)

			// Act
			err := s.DeletePushSubscription(tt.args.ctx, tt.args.id)

			// Assert
			if tt.wantErr {
				assert.Error(t, err)

				if tt.err != nil {
					assert.Contains(t, err.Error(), tt.err.Error())
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
