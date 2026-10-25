# Stage 1: Builder
FROM golang:1.25-alpine@sha256:1ae0735f00daffa3aaf1363a5184c0d2dc55c78e3db4ec70241cdac97bf84b59 AS builder

ENV GOTOOLCHAIN=auto

RUN apk add --no-cache git

WORKDIR /app/engine-go

COPY engine-go/go.mod engine-go/go.sum ./
COPY contracts/ /app/contracts/
RUN go mod download

COPY engine-go/ .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o watink-engine cmd/engine/main.go

# Stage 2: minimal runtime
FROM gcr.io/distroless/static-debian12@sha256:d75cdd72874d4790092fcb1b058493ecf6bb5bf2b2b897045b00ff01d91843f2

WORKDIR /

COPY --from=builder /app/engine-go/watink-engine /watink-engine

ENTRYPOINT ["/watink-engine"]
