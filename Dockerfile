# Stage 1: build
FROM golang:1.24-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /server ./cmd/server/

# Stage 2: minimal runtime image
FROM scratch

COPY --from=builder /server /server

EXPOSE 8080

ENTRYPOINT ["/server"]
