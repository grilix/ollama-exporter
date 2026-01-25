FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o ollama-exporter ./cmd/ollama-exporter

FROM alpine:latest

RUN adduser -D -u 1000 user

EXPOSE 8000

WORKDIR /home/user

COPY --from=builder /app/ollama-exporter .

RUN chown user:user ollama-exporter

USER user

CMD ["./ollama-exporter"]
