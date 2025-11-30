# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o forum-app .

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/forum-app .
COPY --from=builder /app/static ./static

EXPOSE 8082

CMD ["./forum-app"]