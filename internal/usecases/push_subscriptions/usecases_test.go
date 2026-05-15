package push_subscriptions_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DKhorkov/kfc/internal/config"
	"github.com/DKhorkov/kfc/internal/domains"
	pushsubscriptions "github.com/DKhorkov/kfc/internal/usecases/push_subscriptions"
	mockservices "github.com/DKhorkov/kfc/mocks/services"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestUseCases_CreatePushSubscription(t *testing.T) {
	t.Parallel()

	type fields struct {
		mockPushSubscriptionsService func(*mockservices.MockPushSubscriptionsService)
	}

	type args struct {
		ctx          context.Context
		subscription domains.PushSubscription
	}

	now := time.Now()

	expectedSubscription := &domains.PushSubscription{
		ID:            1,
		UserID:        10,
		Endpoint:      "https://push.example.com/sub1",
		EncryptionKey: "test-key",
		Auth:          "test-auth",
		CreatedAt:     now,
	}

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
				mockPushSubscriptionsService: func(ps *mockservices.MockPushSubscriptionsService) {
					ps.EXPECT().
						CreatePushSubscription(gomock.Any(), domains.PushSubscription{
							UserID:        10,
							Endpoint:      "https://push.example.com/sub1",
							EncryptionKey: "test-key",
							Auth:          "test-auth",
						}).
						Return(expectedSubscription, nil)
				},
			},
			args: args{
				ctx: context.Background(),
				subscription: domains.PushSubscription{
					UserID:        10,
					Endpoint:      "https://push.example.com/sub1",
					EncryptionKey: "test-key",
					Auth:          "test-auth",
				},
			},
			want:    expectedSubscription,
			wantErr: false,
		},
		{
			name: "service returns error",
			fields: fields{
				mockPushSubscriptionsService: func(ps *mockservices.MockPushSubscriptionsService) {
					ps.EXPECT().
						CreatePushSubscription(gomock.Any(), gomock.Any()).
						Return(nil, errors.New("database error"))
				},
			},
			args: args{
				ctx: context.Background(),
				subscription: domains.PushSubscription{
					UserID:   10,
					Endpoint: "https://push.example.com/sub1",
				},
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

			mockPushSubscriptionsService := mockservices.NewMockPushSubscriptionsService(ctrl)

			if tt.fields.mockPushSubscriptionsService != nil {
				tt.fields.mockPushSubscriptionsService(mockPushSubscriptionsService)
			}

			u := pushsubscriptions.New(
				mockPushSubscriptionsService,
				config.WebPushConfig{},
				nil,
			)

			// Act
			got, err := u.CreatePushSubscription(tt.args.ctx, tt.args.subscription)

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

func TestUseCases_GetPushSubscriptionsByUserID(t *testing.T) {
	t.Parallel()

	type fields struct {
		mockPushSubscriptionsService func(*mockservices.MockPushSubscriptionsService)
	}

	type args struct {
		ctx    context.Context
		userID uint64
	}

	now := time.Now()

	expectedSubscriptions := []domains.PushSubscription{
		{
			ID:            1,
			UserID:        10,
			Endpoint:      "https://push.example.com/sub1",
			EncryptionKey: "test-key-1",
			Auth:          "test-auth-1",
			CreatedAt:     now,
		},
		{
			ID:            2,
			UserID:        10,
			Endpoint:      "https://push.example.com/sub2",
			EncryptionKey: "test-key-2",
			Auth:          "test-auth-2",
			CreatedAt:     now,
		},
	}

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
				mockPushSubscriptionsService: func(ps *mockservices.MockPushSubscriptionsService) {
					ps.EXPECT().
						GetPushSubscriptionsByUserID(gomock.Any(), uint64(10)).
						Return(expectedSubscriptions, nil)
				},
			},
			args: args{
				ctx:    context.Background(),
				userID: 10,
			},
			want:    expectedSubscriptions,
			wantErr: false,
		},
		{
			name: "service returns error",
			fields: fields{
				mockPushSubscriptionsService: func(ps *mockservices.MockPushSubscriptionsService) {
					ps.EXPECT().
						GetPushSubscriptionsByUserID(gomock.Any(), uint64(999)).
						Return(nil, errors.New("not found"))
				},
			},
			args: args{
				ctx:    context.Background(),
				userID: 999,
			},
			want:    nil,
			wantErr: true,
			err:     errors.New("not found"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			ctrl := gomock.NewController(t)

			mockPushSubscriptionsService := mockservices.NewMockPushSubscriptionsService(ctrl)

			if tt.fields.mockPushSubscriptionsService != nil {
				tt.fields.mockPushSubscriptionsService(mockPushSubscriptionsService)
			}

			u := pushsubscriptions.New(
				mockPushSubscriptionsService,
				config.WebPushConfig{},
				nil,
			)

			// Act
			got, err := u.GetPushSubscriptionsByUserID(tt.args.ctx, tt.args.userID)

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

func TestUseCases_DeletePushSubscription(t *testing.T) {
	t.Parallel()

	type fields struct {
		mockPushSubscriptionsService func(*mockservices.MockPushSubscriptionsService)
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
				mockPushSubscriptionsService: func(ps *mockservices.MockPushSubscriptionsService) {
					ps.EXPECT().
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
			name: "service returns error",
			fields: fields{
				mockPushSubscriptionsService: func(ps *mockservices.MockPushSubscriptionsService) {
					ps.EXPECT().
						DeletePushSubscription(gomock.Any(), uint64(999)).
						Return(errors.New("not found"))
				},
			},
			args: args{
				ctx: context.Background(),
				id:  999,
			},
			wantErr: true,
			err:     errors.New("not found"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			ctrl := gomock.NewController(t)

			mockPushSubscriptionsService := mockservices.NewMockPushSubscriptionsService(ctrl)

			if tt.fields.mockPushSubscriptionsService != nil {
				tt.fields.mockPushSubscriptionsService(mockPushSubscriptionsService)
			}

			u := pushsubscriptions.New(
				mockPushSubscriptionsService,
				config.WebPushConfig{},
				nil,
			)

			// Act
			err := u.DeletePushSubscription(tt.args.ctx, tt.args.id)

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
