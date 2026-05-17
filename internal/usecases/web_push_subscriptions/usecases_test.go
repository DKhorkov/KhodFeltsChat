package web_push_subscriptions_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DKhorkov/kfc/internal/domains"
	pushsubscriptions "github.com/DKhorkov/kfc/internal/usecases/web_push_subscriptions"
	mockservices "github.com/DKhorkov/kfc/mocks/services"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestUseCases_CreateWebPushSubscription(t *testing.T) {
	t.Parallel()

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
		name       string
		setupMocks func(*mockservices.MockWebPushSubscriptionsService)
		args       domains.WebPushSubscription
		want       *domains.WebPushSubscription
		wantErr    bool
		err        error
	}{
		{
			name: "successfully create push subscription",
			setupMocks: func(ps *mockservices.MockWebPushSubscriptionsService) {
				ps.EXPECT().
					CreateWebPushSubscription(gomock.Any(), domains.WebPushSubscription{
						UserID:        10,
						Endpoint:      "https://push.example.com/sub1",
						EncryptionKey: "test-key",
						Auth:          "test-auth",
					}).
					Return(expectedSubscription, nil)
			},
			args: domains.WebPushSubscription{
				UserID:        10,
				Endpoint:      "https://push.example.com/sub1",
				EncryptionKey: "test-key",
				Auth:          "test-auth",
			},
			want:    expectedSubscription,
			wantErr: false,
		},
		{
			name: "service returns error",
			setupMocks: func(ps *mockservices.MockWebPushSubscriptionsService) {
				ps.EXPECT().
					CreateWebPushSubscription(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("database error"))
			},
			args: domains.WebPushSubscription{
				UserID:   10,
				Endpoint: "https://push.example.com/sub1",
			},
			want:    nil,
			wantErr: true,
			err:     errors.New("database error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockService := mockservices.NewMockWebPushSubscriptionsService(ctrl)

			if tt.setupMocks != nil {
				tt.setupMocks(mockService)
			}

			u := pushsubscriptions.New(mockService)

			got, err := u.CreateWebPushSubscription(context.Background(), tt.args)

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

	tests := []struct {
		name       string
		setupMocks func(*mockservices.MockWebPushSubscriptionsService)
		id         uint64
		wantErr    bool
		err        error
	}{
		{
			name: "successfully delete push subscription",
			setupMocks: func(ps *mockservices.MockWebPushSubscriptionsService) {
				ps.EXPECT().
					DeleteWebPushSubscription(gomock.Any(), uint64(1)).
					Return(nil)
			},
			id:      1,
			wantErr: false,
		},
		{
			name: "service returns error",
			setupMocks: func(ps *mockservices.MockWebPushSubscriptionsService) {
				ps.EXPECT().
					DeleteWebPushSubscription(gomock.Any(), uint64(999)).
					Return(errors.New("not found"))
			},
			id:      999,
			wantErr: true,
			err:     errors.New("not found"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockService := mockservices.NewMockWebPushSubscriptionsService(ctrl)

			if tt.setupMocks != nil {
				tt.setupMocks(mockService)
			}

			u := pushsubscriptions.New(mockService)

			err := u.DeleteWebPushSubscription(context.Background(), tt.id)

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
