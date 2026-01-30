# Prometheus Ollama Exporter

This is a proxy for extracting metrics from Ollama and OpenAI compatible endpoints, to export
them to prometheus.

It currently supports the common endpoints for chat and completion, on both streaming and
non-streaming requests, allowing performance and usage monitoring.

## Running

### From source

```
go run ./cmd/ollama-exporter
```

### From binary

Download the binary from the releases, or build it yourself:

```
go build -o ollama-exporter ./cmd/ollama-exporter
```

### From docker

```
docker run --rm -p 8000:8000 -e OLLAMA_HOST ghcr.io/grilix/ollama-exporter:latest
```

## Configuration

The configuration is done via the environment variables:

Name|Default|Description
---|---|---
`OLLAMA_HOST`|`"localhost:11434"`|Ollama host to forward the requests to
`OLLAMA_TIMEOUT`|`"50m"`|Timeout for the requests
`EXPORTER_ADDR`|`""`|IP address where the proxy will bind to
`EXPORTER_PORT`|`8000`|Port on which the proxy will listen at

## Usage

Point your client to the address where the proxy is listening at, for example:

```
OLLAMA_HOST=127.0.0.1:8000 ollama list
```

## Metrics

The metrics are exposed at `/metrics`:

```
curl http://127.0.0.1:8000/metrics
```

NOTE: There are metrics that are only available for the ollama endpoints.

See `cmd/ollama-exporter/main.go` for a list of metrics.
