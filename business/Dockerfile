# ESTÁGIO 1: Builder
FROM golang:1.24-alpine@sha256:8bee1901f1e530bfb4a7850aa7a479d17ae3a18beb6e09064ed54cfd245b7191 AS builder

WORKDIR /app

# Instala dependências necessárias para o build
RUN apk add --no-cache git

# Cache de dependências
COPY go.mod go.sum ./
RUN go mod download

# Copia o código fonte
COPY . .

# Compilação estática do binário Business
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o watink-business cmd/server/main.go

# ESTÁGIO 2: Imagem Final (Distroless para Segurança e Tamanho)
FROM gcr.io/distroless/static-debian12@sha256:d75cdd72874d4790092fcb1b058493ecf6bb5bf2b2b897045b00ff01d91843f2

WORKDIR /

# Copia o binário compilado
COPY --from=builder /app/watink-business /watink-business

# Expõe a porta do servidor Business
EXPOSE 8082

# Variáveis de ambiente padrão
ENV PORT_GO=8082 \
    GIN_MODE=release

# Executa o binário diretamente
ENTRYPOINT ["/watink-business"]
