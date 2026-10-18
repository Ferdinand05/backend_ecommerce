# Tahap 1: Membangun Binary (Kompilasi)
FROM golang:1.26-alpine AS builder
WORKDIR /app

# Salin file modul dan unduh dependencies terlebih dahulu (untuk caching)
COPY go.mod go.sum ./
RUN go mod download

# Salin seluruh kode proyek
COPY . .

# Compile aplikasi menjadi satu file binary bernama 'api-server'
RUN CGO_ENABLED=0 GOOS=linux go build -o api-server ./cmd/main.go

# Tahap 2: Menjalankan Aplikasi di Image Bersih dan Ringan
FROM alpine:latest
WORKDIR /app

# Salin file binary hasil kompilasi dari Tahap 1
COPY --from=builder /app/api-server .

EXPOSE 8080

# Jalankan file binary langsung tanpa butuh instalasi Go
CMD ["./api-server"]