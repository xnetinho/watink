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

# Stage 2: runtime. Precisa do ffmpeg/ffprobe: o engine gera miniatura, dimensões, duração e forma de onda (e converte
# áudio para Opus; poppler-utils dá a miniatura e o nº de páginas de PDF) para que as mídias saiam como as de um cliente oficial do WhatsApp (pacote internal/mediameta).
# Alpine em vez de distroless só por causa do ffmpeg. O usuário continua o mesmo de antes (root): o volume do outbox já
# pertence a ele, e trocar de UID faria o engine perder o acesso ao outbox existente (hardening fica para outra mudança).
FROM alpine:3.22@sha256:2e0b81d60597f449baa56dc25d355367869d2dca4cd57e866f1e2b52af3eb55f

RUN apk add --no-cache ffmpeg poppler-utils ca-certificates tzdata

WORKDIR /

COPY --from=builder /app/engine-go/watink-engine /watink-engine

ENTRYPOINT ["/watink-engine"]
