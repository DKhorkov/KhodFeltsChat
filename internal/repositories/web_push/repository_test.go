package web_push_test

import (
	"testing"

	"github.com/DKhorkov/kfc/internal/config"
	web_push "github.com/DKhorkov/kfc/internal/repositories/web_push"
	"github.com/DKhorkov/libs/logging/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestNew(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		webPushConfig config.WebPushConfig
	}{
		{
			name: "create repository with valid config",
			webPushConfig: config.WebPushConfig{
				VAPIDPublicKey:  "test-public-key",
				VAPIDPrivateKey: "test-private-key",
				VAPIDContact:    "mailto:test@example.com",
				TTL:             60,
			},
		},
		{
			name:          "create repository with empty config",
			webPushConfig: config.WebPushConfig{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			logger := mocks.NewMockLogger(ctrl)
			repo := web_push.New(tt.webPushConfig, logger)

			assert.NotNil(t, repo)
		})
	}
}
