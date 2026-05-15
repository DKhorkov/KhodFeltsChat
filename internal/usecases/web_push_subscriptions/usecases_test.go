package web_push_subscriptions_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DKhorkov/kfc/internal/config"
	"github.com/DKhorkov/kfc/internal/domains"
	pushsubscriptions "github.com/DKhorkov/kfc/internal/usecases/web_push_subscriptions"
	mockservices "github.com/DKhorkov/kfc/mocks/services"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestUseCases_CreateWebPushSubscription(t *testing.T) {
	t.Parallel()

	type fields struct {
		mockWebPushSubscriptionsService func(*mockservices.MockWebPushSubscriptionsService)
	}

	type args struct {
		ctx          context.Context
		subscription domains.WebPushSubscription
	}

	now := time.Now()

	expectedSubscription := &domains.WebPushSubscription{
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
		want    *domains.WebPushSubscription
		wantErr bool
		err     error
	}{
		{
			name: "successfully create push subscription",
			fields: fields{
				mockWebPushSubscriptionsService: func(ps *mockservices.MockWebPushSubscriptionsService) {
					ps.EXPECT().
						CreateWebPushSubscription(gomock.Any(), domains.WebPushSubscription{
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
				subscription: domains.WebPushSubscription{
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
				mockWebPushSubscriptionsService: func(ps *mockservices.MockWebPushSubscriptionsService) {
					ps.EXPECT().
						CreateWebPushSubscription(gomock.Any(), gomock.Any()).
						Return(nil, errors.New("database error"))
				},
			},
			args: args{
				ctx: context.Background(),
				subscription: domains.WebPushSubscription{
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

			mockWebPushSubscriptionsService := mockservices.NewMockWebPushSubscriptionsService(ctrl)

			if tt.fields.mockWebPushSubscriptionsService != nil {
				tt.fields.mockWebPushSubscriptionsService(mockWebPushSubscriptionsService)
			}

			u := pushsubscriptions.New(
				mockWebPushSubscriptionsService,
				config.WebPushConfig{},
				nil,
			)

			// Act
			got, err := u.CreateWebPushSubscription(tt.args.ctx, tt.args.subscription)

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

func TestUseCases_GetWebPushSubscriptionsByUserID(t *testing.T) {
	t.Parallel()

	type fields struct {
		mockWebPushSubscriptionsService func(*mockservices.MockWebPushSubscriptionsService)
	}

	type args struct {
		ctx    context.Context
		userID uint64
	}

	now := time.Now()

	expectedSubscriptions := []domains.WebPushSubscription{
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
		want    []domains.WebPushSubscription
		wantErr bool
		err     error
	}{
		{
			name: "successfully get push subscriptions",
			fields: fields{
				mockWebPushSubscriptionsService: func(ps *mockservices.MockWebPushSubscriptionsService) {
					ps.EXPECT().
						GetWebPushSubscriptionsByUserID(gomock.Any(), uint64(10)).
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
				mockWebPushSubscriptionsService: func(ps *mockservices.MockWebPushSubscriptionsService) {
					ps.EXPECT().
						GetWebPushSubscriptionsByUserID(gomock.Any(), uint64(999)).
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

			mockWebPushSubscriptionsService := mockservices.NewMockWebPushSubscriptionsService(ctrl)

			if tt.fields.mockWebPushSubscriptionsService != nil {
				tt.fields.mockWebPushSubscriptionsService(mockWebPushSubscriptionsService)
			}

			u := pushsubscriptions.New(
				mockWebPushSubscriptionsService,
				config.WebPushConfig{},
				nil,
			)

			// Act
			got, err := u.GetWebPushSubscriptionsByUserID(tt.args.ctx, tt.args.userID)

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

func TestUseCases_DeleteWebPushSubscription(t *testing.T) {
	t.Parallel()

	type fields struct {
		mockWebPushSubscriptionsService func(*mockservices.MockWebPushSubscriptionsService)
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
				mockWebPushSubscriptionsService: func(ps *mockservices.MockWebPushSubscriptionsService) {
					ps.EXPECT().
						DeleteWebPushSubscription(gomock.Any(), uint64(1)).
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
				mockWebPushSubscriptionsService: func(ps *mockservices.MockWebPushSubscriptionsService) {
					ps.EXPECT().
						DeleteWebPushSubscription(gomock.Any(), uint64(999)).
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

			mockWebPushSubscriptionsService := mockservices.NewMockWebPushSubscriptionsService(ctrl)

			if tt.fields.mockWebPushSubscriptionsService != nil {
				tt.fields.mockWebPushSubscriptionsService(mockWebPushSubscriptionsService)
			}

			u := pushsubscriptions.New(
				mockWebPushSubscriptionsService,
				config.WebPushConfig{},
				nil,
			)

			// Act
			err := u.DeleteWebPushSubscription(tt.args.ctx, tt.args.id)

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
