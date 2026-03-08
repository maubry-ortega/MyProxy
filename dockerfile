# Phase 1
FROM golang:1.25-alpine AS builder

WORKDIR /app

# 1️⃣ Descargar dependencias primero (capa cacheable)
COPY go.mod go.sum ./
RUN go mod download

# 2️⃣ Copiar código después
COPY . .

# 3️⃣ Build optimizado
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
    -trimpath \
    -ldflags="-s -w" \
    -o myproxy ./cmd/myproxy/main.go

# Phase 2
FROM scratch

COPY --from=builder /app/myproxy /myproxy

EXPOSE 80 443 8080
ENTRYPOINT ["/myproxy"]