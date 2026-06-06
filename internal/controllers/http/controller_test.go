package http_test

import (
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/DKhorkov/kfc/internal/config"
	httpcontroller "github.com/DKhorkov/kfc/internal/controllers/http"
	mockupgrader "github.com/DKhorkov/kfc/mocks/upgrader"
	mockusecases "github.com/DKhorkov/kfc/mocks/usecases"
	"github.com/DKhorkov/libs/logging/mocks"
	"github.com/DKhorkov/libs/security"
	"github.com/DKhorkov/libs/tracing"
	tracingmocks "github.com/DKhorkov/libs/tracing/mocks"
	"go.uber.org/mock/gomock"
)

func TestMain(m *testing.M) {
	if err := os.Chdir("../../.."); err != nil {
		panic("failed to chdir to project root: " + err.Error())
	}

	os.Exit(m.Run())
}

func newTestController(t *testing.T) *httpcontroller.Controller {
	t.Helper()

	ctrl := gomock.NewController(t)

	c, err := httpcontroller.New(
		config.HTTPConfig{
			Host:         "localhost",
			Port:         "0",
			IdleTimeout:  time.Second,
			ReadTimeout:  time.Second,
			WriteTimeout: time.Second,
		},
		config.CORSConfig{},
		config.DocsConfig{
			Dir:      "api",
			Filepath: "api/swagger.json",
		},
		config.CookiesConfig{},
		config.NATSConfig{},
		mockusecases.NewMockUsersUseCases(ctrl),
		mockusecases.NewMockAuthUseCases(ctrl),
		mockusecases.NewMockChatsUseCases(ctrl),
		mockusecases.NewMockMessagesUseCases(ctrl),
		mockusecases.NewMockSettingsUseCases(ctrl),
		mockusecases.NewMockWebPushSubscriptionsUseCases(ctrl),
		mockusecases.NewMockFileStorageUseCases(ctrl),
		mocks.NewMockLogger(ctrl),
		tracingmocks.NewMockProvider(ctrl),
		mockupgrader.NewMockUpgrader(ctrl),
		nil,
		tracing.SpanConfig{},
		security.Config{},
		nil,
		"",
	)
	if err != nil {
		t.Fatalf("Failed to create controller: %v", err)
	}

	return c
}

func TestNew_Success(t *testing.T) {
	t.Parallel()

	c := newTestController(t)

	if c == nil {
		t.Fatal("Expected non-nil controller")
	}
}

func TestNew_DifferentPorts(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)

	ports := []string{"0", "8081", "9090"}
	for _, port := range ports {
		c, err := httpcontroller.New(
			config.HTTPConfig{
				Host:         "localhost",
				Port:         port,
				IdleTimeout:  time.Second,
				ReadTimeout:  time.Second,
				WriteTimeout: time.Second,
			},
			config.CORSConfig{},
			config.DocsConfig{
				Dir:      "api",
				Filepath: "api/swagger.json",
			},
			config.CookiesConfig{},
			config.NATSConfig{},
			mockusecases.NewMockUsersUseCases(ctrl),
			mockusecases.NewMockAuthUseCases(ctrl),
			mockusecases.NewMockChatsUseCases(ctrl),
			mockusecases.NewMockMessagesUseCases(ctrl),
			mockusecases.NewMockSettingsUseCases(ctrl),
			mockusecases.NewMockWebPushSubscriptionsUseCases(ctrl),
			mockusecases.NewMockFileStorageUseCases(ctrl),
			mocks.NewMockLogger(ctrl),
			tracingmocks.NewMockProvider(ctrl),
			mockupgrader.NewMockUpgrader(ctrl),
			nil,
			tracing.SpanConfig{},
			security.Config{},
			nil,
			"",
		)
		if err != nil {
			t.Errorf("Failed to create controller with port %s: %v", port, err)
		}

		if c == nil {
			t.Errorf("Expected non-nil controller for port %s", port)
		}
	}
}

func TestRunAndStop(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	mockLogger := mocks.NewMockLogger(ctrl)
	mockLogger.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()

	c, err := httpcontroller.New(
		config.HTTPConfig{
			Host:         "127.0.0.1",
			Port:         "0",
			IdleTimeout:  time.Second,
			ReadTimeout:  time.Second,
			WriteTimeout: time.Second,
		},
		config.CORSConfig{},
		config.DocsConfig{
			Dir:      "api",
			Filepath: "api/swagger.json",
		},
		config.CookiesConfig{},
		config.NATSConfig{},
		mockusecases.NewMockUsersUseCases(ctrl),
		mockusecases.NewMockAuthUseCases(ctrl),
		mockusecases.NewMockChatsUseCases(ctrl),
		mockusecases.NewMockMessagesUseCases(ctrl),
		mockusecases.NewMockSettingsUseCases(ctrl),
		mockusecases.NewMockWebPushSubscriptionsUseCases(ctrl),
		mockusecases.NewMockFileStorageUseCases(ctrl),
		mockLogger,
		tracingmocks.NewMockProvider(ctrl),
		mockupgrader.NewMockUpgrader(ctrl),
		nil,
		tracing.SpanConfig{},
		security.Config{},
		nil,
		"",
	)
	if err != nil {
		t.Fatalf("Failed to create controller: %v", err)
	}

	done := make(chan struct{})

	go func() {
		c.Run()
		close(done)
	}()

	// Даём серверу время на запуск.
	time.Sleep(50 * time.Millisecond)
	c.Stop()

	select {
	case <-done:
		// Сервер успешно остановлен.
	case <-time.After(3 * time.Second):
		t.Fatal("Server did not stop within timeout")
	}
}

func TestNew_WithCORS(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)

	c, err := httpcontroller.New(
		config.HTTPConfig{
			Host:         "localhost",
			Port:         "0",
			IdleTimeout:  time.Second,
			ReadTimeout:  time.Second,
			WriteTimeout: time.Second,
		},
		config.CORSConfig{
			AllowedOrigins: []string{"http://localhost:3000"},
			AllowedMethods: []string{http.MethodGet, http.MethodPost},
			AllowedHeaders: []string{"Content-Type"},
			MaxAge:         300, AllowCredentials: true,
		},
		config.DocsConfig{
			Dir:      "api",
			Filepath: "api/swagger.json",
		},
		config.CookiesConfig{},
		config.NATSConfig{},
		mockusecases.NewMockUsersUseCases(ctrl),
		mockusecases.NewMockAuthUseCases(ctrl),
		mockusecases.NewMockChatsUseCases(ctrl),
		mockusecases.NewMockMessagesUseCases(ctrl),
		mockusecases.NewMockSettingsUseCases(ctrl),
		mockusecases.NewMockWebPushSubscriptionsUseCases(ctrl),
		mockusecases.NewMockFileStorageUseCases(ctrl),
		mocks.NewMockLogger(ctrl),
		tracingmocks.NewMockProvider(ctrl),
		mockupgrader.NewMockUpgrader(ctrl),
		nil,
		tracing.SpanConfig{},
		security.Config{},
		[]string{"password", "token"},
		"",
	)
	if err != nil {
		t.Fatalf("Failed to create controller: %v", err)
	}

	if c == nil {
		t.Fatal("Expected non-nil controller")
	}
}
