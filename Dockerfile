FROM golang:1.24 AS builder

WORKDIR /storyhub-build

RUN apt-get update && \
    apt-get install -y --no-install-recommends make ca-certificates && \
    update-ca-certificates && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN make build

FROM scratch

WORKDIR /app

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /storyhub-build/main /app/

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD curl --fail http://localhost:8000/api/v1/health || exit 1

CMD ["./main"]
