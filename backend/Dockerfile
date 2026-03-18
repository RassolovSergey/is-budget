FROM golang:1.23.12-alpine3.21 AS builder
WORKDIR /src

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app

COPY --from=builder /out/api /app/api
COPY internal/db/migrations /app/migrations
COPY openapi.yaml /app/openapi.yaml

EXPOSE 8080

CMD ["/app/api"]
