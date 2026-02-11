FROM golang:1.23-alpine AS builder

ARG VERSION="dev"
ARG COMMIT_SHA="unknown"
ENV BUILD_VERSION=${VERSION}
ENV BUILD_SHA=${COMMIT_SHA}

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY --parents \
  main.go \
  internal \
  cmd \
  pkg \
  .

COPY internal/ internal

RUN CGO_ENABLED=0 GOOS=linux \
      go build -a -installsuffix cgo \
      -ldflags "-X main.exporterVersion=${VERSION} -X main.exporterSha=${COMMIT_SHA}" \
      -o ollama-exporter \
      .

FROM alpine:latest

RUN adduser -D -u 1000 user

EXPOSE 8000

WORKDIR /home/user

COPY --from=builder /app/ollama-exporter .

COPY README.md LICENSE .

RUN chown user:user ollama-exporter

USER user

CMD ["./ollama-exporter"]
