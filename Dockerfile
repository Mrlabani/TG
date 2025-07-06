FROM golang:1.21-alpine3.18 AS builder

RUN apk update && apk upgrade --available && apk add --no-cache git

WORKDIR /app
COPY . .

RUN go mod tidy && \
    CGO_ENABLED=0 go build -o /app/fsb -ldflags="-w -s" ./cmd/fsb

# Minimal final image
FROM scratch

COPY --from=builder /app/fsb /app/fsb

# Set default port (optional)
ARG PORT=8080
EXPOSE ${PORT}

ENTRYPOINT ["/app/fsb", "run"]
