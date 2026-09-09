# Build context must be the repo root (services/analytic_service depends on ../../shared
# via a local `replace` directive in go.mod, so the shared module source must be
# present in the build context too).
FROM golang:1.26-alpine AS builder

WORKDIR /workspace

COPY shared/ shared/
COPY services/analytic_service/ services/analytic_service/

WORKDIR /workspace/services/analytic_service
RUN CGO_ENABLED=0 go build -o /out/analytic_service ./cmd

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /out/analytic_service .

EXPOSE 8085
ENTRYPOINT ["./analytic_service"]
