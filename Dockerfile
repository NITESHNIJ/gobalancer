FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/gobalancer ./cmd/gobalancer

FROM scratch
COPY --from=builder /bin/gobalancer /gobalancer
COPY --from=builder /app/config/config.yaml /config/config.yaml
EXPOSE 8080 9001
ENTRYPOINT ["/gobalancer", "-config", "/config/config.yaml"]
