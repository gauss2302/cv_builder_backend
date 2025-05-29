FROM golang:1.24-alpine AS builder

RUN apk update && apk add --no-cache git ca-certificates && update-ca-certificates

WORKDIR /app

LABEL authors="nikitashilov"

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-s -w" -o cv_builder ./cmd/server/main.go

# Final stage
FROM alpine:3.21

RUN echo "https://dl-cdn.alpinelinux.org/alpine/v3.21/community" >> /etc/apk/repositories \
 && apk update \
 && apk add --no-cache ca-certificates tzdata \
 && update-ca-certificates

RUN adduser -D -g '' appuser
USER appuser

WORKDIR /app

COPY --from=builder /app/cv_builder .
COPY --from=builder /app/migrations ./migrations

EXPOSE 8080
CMD ["./cv_builder"]
