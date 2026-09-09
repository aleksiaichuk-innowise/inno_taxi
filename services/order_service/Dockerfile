# Build context must be the repo root (services/order_service depends on ../../shared
# via a local `replace` directive in go.mod, so the shared module source must be
# present in the build context too).
FROM golang:1.26-alpine AS builder

WORKDIR /workspace

COPY shared/ shared/
COPY services/order_service/ services/order_service/

WORKDIR /workspace/services/order_service
RUN CGO_ENABLED=0 go build -o /out/order_service ./cmd

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /out/order_service .

EXPOSE 8080 9082
ENTRYPOINT ["./order_service"]
