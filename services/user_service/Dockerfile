# Build context must be the repo root (services/user_service depends on ../../shared
# via a local `replace` directive in go.mod, so the shared module source must be
# present in the build context too).
FROM golang:1.26-alpine AS builder

WORKDIR /workspace

COPY shared/ shared/
COPY services/user_service/ services/user_service/

WORKDIR /workspace/services/user_service
RUN CGO_ENABLED=0 go build -o /out/user_service ./cmd

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /out/user_service .

EXPOSE 8080
ENTRYPOINT ["./user_service"]
