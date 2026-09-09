# Build context must be the repo root (services/driver_service depends on ../../shared
# via a local `replace` directive in go.mod, so the shared module source must be
# present in the build context too).
FROM golang:1.26-alpine AS builder

WORKDIR /workspace

COPY shared/ shared/
COPY services/driver_service/ services/driver_service/

WORKDIR /workspace/services/driver_service
RUN CGO_ENABLED=0 go build -o /out/driver_service ./cmd

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /out/driver_service .

EXPOSE 8081
ENTRYPOINT ["./driver_service"]
