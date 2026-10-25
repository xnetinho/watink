# Stage 1: Builder
FROM golang:1.25-alpine AS builder

ENV GOTOOLCHAIN=auto

RUN apk add --no-cache git

WORKDIR /app/engine-go

COPY engine-go/go.mod engine-go/go.sum ./
COPY contracts/ /app/contracts/
RUN go mod download

COPY engine-go/ .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o watink-engine cmd/engine/main.go

# Stage 2: minimal runtime
FROM gcr.io/distroless/static-debian12

WORKDIR /

COPY --from=builder /app/engine-go/watink-engine /watink-engine

ENTRYPOINT ["/watink-engine"]
