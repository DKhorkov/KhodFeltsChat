# Move Handlers to api/ Subdirectory — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Перенести хэндлеры `auth`, `users`, `ws`, `chats`, `messages` в `internal/controllers/http/handlers/api/`, чтобы подготовить структуру для будущего `web/` пакета.

**Architecture:** Оркестратор `handlers/setup.go` создаёт subrouter с префиксом `/api` и делегирует `api.SetupHandlers(...)`. Shared пакеты (`common`, `default`, `not_allowed`, `docs`) остаются на уровне `handlers/`. URL-константы переезжают в `api/setup.go`.

**Tech Stack:** Go 1.24, gorilla/mux, существующая кодовая база KFC.

---

### Task 1: Создать директорию `api/` и переместить пакеты хэндлеров

**Files:**
- Move: `internal/controllers/http/handlers/auth/` → `internal/controllers/http/handlers/api/auth/`
- Move: `internal/controllers/http/handlers/users/` → `internal/controllers/http/handlers/api/users/`
- Move: `internal/controllers/http/handlers/ws/` → `internal/controllers/http/handlers/api/ws/`
- Move: `internal/controllers/http/handlers/chats/` → `internal/controllers/http/handlers/api/chats/`
- Move: `internal/controllers/http/handlers/messages/` → `internal/controllers/http/handlers/api/messages/`

- [ ] **Step 1: Создать директорию api и переместить пакеты**

```bash
mkdir -p internal/controllers/http/handlers/api
mv internal/controllers/http/handlers/auth internal/controllers/http/handlers/api/auth
mv internal/controllers/http/handlers/users internal/controllers/http/handlers/api/users
mv internal/controllers/http/handlers/ws internal/controllers/http/handlers/api/ws
mv internal/controllers/http/handlers/chats internal/controllers/http/handlers/api/chats
mv internal/controllers/http/handlers/messages internal/controllers/http/handlers/api/messages
```

- [ ] **Step 2: Убедиться, что структура корректна**

```bash
find internal/controllers/http/handlers/api -type d | sort
```

Expected:
```
internal/controllers/http/handlers/api
internal/controllers/http/handlers/api/auth
internal/controllers/http/handlers/api/auth/change_password
internal/controllers/http/handlers/api/auth/forget_password
internal/controllers/http/handlers/api/auth/login
internal/controllers/http/handlers/api/auth/logout
internal/controllers/http/handlers/api/auth/refresh_tokens
internal/controllers/http/handlers/api/auth/register
internal/controllers/http/handlers/api/auth/send_forget_password_message
internal/controllers/http/handlers/api/auth/send_verify_email_message
internal/controllers/http/handlers/api/auth/verify_email
internal/controllers/http/handlers/api/chats
internal/controllers/http/handlers/api/chats/create
internal/controllers/http/handlers/api/chats/user_chats
internal/controllers/http/handlers/api/messages
internal/controllers/http/handlers/api/messages/chat_messages
internal/controllers/http/handlers/api/users
internal/controllers/http/handlers/api/users/me
internal/controllers/http/handlers/api/users/update
internal/controllers/http/handlers/api/users/user_by_id
internal/controllers/http/handlers/api/users/users
internal/controllers/http/handlers/api/ws
```

---

### Task 2: Создать `api/setup.go` с URL-константами и функцией SetupHandlers

**Files:**
- Create: `internal/controllers/http/handlers/api/setup.go`

- [ ] **Step 1: Создать файл `api/setup.go`**

```go
package api

import (
	"fmt"
	"net/http"

	"github.com/DKhorkov/kfc/internal/config"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/auth/change_password"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/auth/forget_password"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/auth/login"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/auth/logout"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/auth/refresh_tokens"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/auth/register"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/auth/send_forget_password_message"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/auth/send_verify_email_message"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/auth/verify_email"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/chats/create"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/chats/user_chats"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/messages/chat_messages"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/users/me"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/users/update"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/users/user_by_id"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/users/users"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/ws"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/common"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/libs/logging"
	"github.com/gorilla/mux"
)

const (
	SessionsURL = "/sessions"

	UsersURL                  = "/users"
	MeURL                     = UsersURL + "/me"
	GetUserByIDURL            = UsersURL + "/{%s}"
	PasswordURL               = UsersURL + "/password"
	ChangePasswordURL         = PasswordURL + "/change"
	SendForgetPasswordURL     = PasswordURL + "/forget"
	ForgetPasswordURL         = SendForgetPasswordURL + "/{%s}"
	SendVerifyEmailMessageURL = UsersURL + "/email/verify"
	VerifyEmailURL            = SendVerifyEmailMessageURL + "/{%s}"

	WebsocketURL = "/ws"

	ChatsURL           = "/chats"
	GetChatMessagesURL = ChatsURL + "/{%s}/messages"
)

func SetupHandlers(
	apiMux *mux.Router,
	cookiesConfig config.CookiesConfig,
	usersUseCases interfaces.UsersUseCases,
	authUseCases interfaces.AuthUseCases,
	chatsUseCases interfaces.ChatsUseCases,
	messagesUseCases interfaces.MessagesUseCases,
	logger logging.Logger,
	upgrader interfaces.Upgrader,
) {
	getMux := apiMux.Methods(http.MethodGet).Subrouter()
	getMux.Handle(UsersURL, users.Handler(usersUseCases))
	getMux.Handle(MeURL, me.Handler(usersUseCases))
	getMux.Handle(
		fmt.Sprintf(GetUserByIDURL, common.IDRouteKey),
		user_by_id.Handler(usersUseCases),
	)

	websocketHandler := ws.New(
		upgrader,
		usersUseCases,
		chatsUseCases,
		messagesUseCases,
		logger,
	)

	getMux.Handle(WebsocketURL, http.HandlerFunc(websocketHandler.Handle))
	getMux.Handle(ChatsURL, user_chats.Handler(chatsUseCases))
	getMux.Handle(
		fmt.Sprintf(GetChatMessagesURL, common.IDRouteKey),
		chat_messages.Handler(messagesUseCases),
	)
	getMux.Handle(
		fmt.Sprintf(VerifyEmailURL, verify_email.TokenRouteKey),
		verify_email.Handler(authUseCases),
	)

	postMux := apiMux.Methods(http.MethodPost).Subrouter()
	postMux.Handle(UsersURL, register.Handler(authUseCases))
	postMux.Handle(SessionsURL, login.Handler(authUseCases, cookiesConfig))
	postMux.Handle(ChangePasswordURL, change_password.Handler(authUseCases))
	postMux.Handle(
		SendVerifyEmailMessageURL,
		send_verify_email_message.Handler(authUseCases),
	)
	postMux.Handle(
		fmt.Sprintf(ForgetPasswordURL, forget_password.TokenRouteKey),
		forget_password.Handler(authUseCases),
	)
	postMux.Handle(
		SendForgetPasswordURL,
		send_forget_password_message.Handler(authUseCases),
	)
	postMux.Handle(ChatsURL, create.Handler(chatsUseCases))

	putMux := apiMux.Methods(http.MethodPut).Subrouter()
	putMux.Handle(MeURL, update.Handler(usersUseCases))
	putMux.Handle(SessionsURL, refresh_tokens.Handler(authUseCases, cookiesConfig))

	deleteMux := apiMux.Methods(http.MethodDelete).Subrouter()
	deleteMux.Handle(SessionsURL, logout.Handler(authUseCases))
}
```

---

### Task 3: Переписать `handlers/setup.go` как оркестратор

**Files:**
- Modify: `internal/controllers/http/handlers/setup.go`

- [ ] **Step 1: Заменить содержимое `handlers/setup.go`**

```go
package handlers

import (
	"fmt"
	"net/http"

	"github.com/DKhorkov/kfc/internal/config"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/api"
	default_handler "github.com/DKhorkov/kfc/internal/controllers/http/handlers/default"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/docs"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/not_allowed"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/libs/logging"
	metricsmiddleware "github.com/DKhorkov/libs/middlewares/http/metrics"
	"github.com/go-openapi/runtime/middleware"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	APIPrefix  = "/api"
	SwaggerURL = "/%s"
)

func SetupHandlers(
	rootMux *mux.Router,
	docsConfig config.DocsConfig,
	cookiesConfig config.CookiesConfig,
	usersUseCases interfaces.UsersUseCases,
	authUseCases interfaces.AuthUseCases,
	chatsUseCases interfaces.ChatsUseCases,
	messagesUseCases interfaces.MessagesUseCases,
	logger logging.Logger,
	upgrader interfaces.Upgrader,
) {
	rootMux.NotFoundHandler = http.HandlerFunc(default_handler.Handler)
	rootMux.MethodNotAllowedHandler = http.HandlerFunc(not_allowed.Handler)

	// Metrics:
	rootMux.Methods(http.MethodGet).Subrouter().Handle(
		metricsmiddleware.MetricsURLPath,
		promhttp.Handler(),
	)

	// Docs (Swagger):
	swaggerURL := fmt.Sprintf(SwaggerURL, docsConfig.Filepath)
	opts := middleware.RedocOpts{
		SpecURL: swaggerURL,
	}
	sh := middleware.Redoc(opts, nil)
	getMux := rootMux.Methods(http.MethodGet).Subrouter()
	getMux.Handle(docs.URL, sh)
	getMux.Handle(swaggerURL, http.FileServer(http.Dir(docsConfig.Dir)))

	// API subrouter:
	apiMux := rootMux.PathPrefix(APIPrefix).Subrouter()
	api.SetupHandlers(
		apiMux,
		cookiesConfig,
		usersUseCases,
		authUseCases,
		chatsUseCases,
		messagesUseCases,
		logger,
		upgrader,
	)
}
```

---

### Task 4: Обновить `controller.go` — импорты и URL-константы с префиксом `/api`

**Files:**
- Modify: `internal/controllers/http/controller.go`

- [ ] **Step 1: Обновить импорты и ссылки на константы**

Заменить импорт:
```go
// Было:
"github.com/DKhorkov/kfc/internal/controllers/http/handlers"
"github.com/DKhorkov/kfc/internal/controllers/http/handlers/auth/login"

// Стало:
"github.com/DKhorkov/kfc/internal/controllers/http/handlers"
"github.com/DKhorkov/kfc/internal/controllers/http/handlers/api"
"github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/auth/login"
```

Обновить все ссылки на URL-константы в auth middleware (добавить `handlers.APIPrefix +` перед каждым путём и заменить `handlers.` на `api.` для URL-констант):

```go
authmiddleware.Middleware(
    login.AccessTokenCookieName,
    securityConfig,
    []authmiddleware.IgnoreURL{
        {
            Path:    regexp.MustCompile(`^` + docs.URL + `$`),
            Methods: []string{http.MethodGet},
        },
        {
            Path: regexp.MustCompile(
                `^` + fmt.Sprintf(handlers.SwaggerURL, docsConfig.Filepath) + `$`,
            ),
            Methods: []string{http.MethodGet},
        },
        {
            Path:    regexp.MustCompile(`^` + handlers.APIPrefix + api.SessionsURL + `$`),
            Methods: []string{http.MethodPost, http.MethodPut},
        },
        {
            Path: regexp.MustCompile(
                `^` + handlers.APIPrefix + api.UsersURL + `(?:\?[^ ]*)?$`,
            ),
            Methods: []string{http.MethodGet},
        },
        {
            Path:    regexp.MustCompile(`^` + handlers.APIPrefix + api.UsersURL + `$`),
            Methods: []string{http.MethodPost},
        },
        {
            Path: regexp.MustCompile(
                `^` + handlers.APIPrefix + strings.ReplaceAll(api.GetUserByIDURL, "{%s}", "") + `(\d+)$`,
            ),
            Methods: []string{http.MethodGet},
        },
        {
            Path: regexp.MustCompile(
                `^` + handlers.APIPrefix + strings.ReplaceAll(api.ForgetPasswordURL, "{%s}", "") + `(.+)$`,
            ),
            Methods: []string{http.MethodPost},
        },
        {
            Path:    regexp.MustCompile(`^` + handlers.APIPrefix + api.SendForgetPasswordURL + `$`),
            Methods: []string{http.MethodPost},
        },
        {
            Path:    regexp.MustCompile(`^` + handlers.APIPrefix + api.SendVerifyEmailMessageURL + `$`),
            Methods: []string{http.MethodPost},
        },
        {
            Path: regexp.MustCompile(
                `^` + handlers.APIPrefix + strings.ReplaceAll(api.VerifyEmailURL, "{%s}", "") + `(.+)$`,
            ),
            Methods: []string{http.MethodGet},
        },
    }...,
),
```

---

### Task 5: Обновить импорты внутри перенесённых хэндлеров

**Files:**
- Modify: все `handler.go` и `handler_test.go` в `internal/controllers/http/handlers/api/`

- [ ] **Step 1: Найти и заменить пути импортов во всех перенесённых файлах**

Выполнить замену во всех `.go` файлах внутри `internal/controllers/http/handlers/api/`:

```bash
find internal/controllers/http/handlers/api -name "*.go" -exec grep -l "github.com/DKhorkov/kfc/internal/controllers/http/handlers/" {} \;
```

Для каждого найденного файла заменить:
```
github.com/DKhorkov/kfc/internal/controllers/http/handlers/common
```
→ оставить как есть (common остался на месте)

Других перекрёстных импортов между перенесёнными пакетами нет — каждый хэндлер импортирует только:
- `common` (остаётся на месте, путь не меняется)
- пакеты из `internal/domains`, `internal/errors`, `internal/interfaces`, `internal/controllers/http/mappers/`, `internal/controllers/http/schemas/` — пути не изменились

Единственный файл с внутренним импортом на уровне handlers — `ws/ws.go`, который импортирует:
```
github.com/DKhorkov/kfc/internal/controllers/http/mappers/messages
```
Этот путь не меняется.

- [ ] **Step 2: Проверить компиляцию**

```bash
go build ./...
```

Expected: успешная компиляция без ошибок.

---

### Task 6: Запустить тесты

**Files:** нет изменений

- [ ] **Step 1: Запустить все тесты**

```bash
go test ./...
```

Expected: все тесты проходят.

---

### Task 7: Commit

- [ ] **Step 1: Зафиксировать изменения**

```bash
git add internal/controllers/http/handlers/api/ internal/controllers/http/handlers/setup.go internal/controllers/http/controller.go
git commit -m "refactor: move API handlers to handlers/api/ with subrouter prefix"
```
