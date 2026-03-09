# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copia dependências primeiro (cache de camadas Docker)
COPY go.mod go.sum ./
RUN go mod download

# Copia o código fonte e compila
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server/

# Runtime stage — imagem mínima
FROM alpine:3.20

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copia apenas o binário compilado
COPY --from=builder /app/server .

EXPOSE 8080

CMD ["./server"]
