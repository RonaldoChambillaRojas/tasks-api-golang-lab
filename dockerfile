# Stage 1: compilar el binario
FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# CGO_ENABLED=0 = sin dependencias de C
# GOOS=linux GOARCH=amd64 = compilado para Linux x64
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-w -s" -o tasks-api .

# Stage 2: imagen final vacía
FROM scratch

COPY --from=builder /app/tasks-api /tasks-api

EXPOSE 8080

ENTRYPOINT ["/tasks-api"]