# https://habr.com/ru/companies/otus/articles/660301/
FROM golang:alpine AS builder
LABEL authors='dkhorkov'

WORKDIR /build

COPY . .

RUN go build -o main ./cmd/main.go

FROM alpine AS runner

WORKDIR /app

COPY .env .
COPY ./api ./api

COPY --from=builder /build/main /app/main

CMD ["./main"]
