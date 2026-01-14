# First stage: build the application
# Docker will pull the version matching our build host.
FROM --platform=$BUILDPLATFORM golang:1.25.5-alpine3.23 AS builder

ARG TARGETARCH
WORKDIR /app

COPY go.mod go.sum ./
COPY src/ src/

# $TARGETARCH so the binary matches the destination, not necessarily the build machine
RUN CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH go build -o web src/cmd/web/*.go && \
    CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH go build -o migrate src/cmd/migrate/*.go

# Second stage: runtime
# Use generic alpine image. Docker will automatically pull the arm64 version when necessary.
FROM alpine:3.23

RUN addgroup -S harmonygroup && adduser -S harmonyuser -G harmonygroup
WORKDIR /app
RUN apk --no-cache add ca-certificates tzdata

COPY --from=builder --chown=harmonyuser:harmonygroup /app/web .
COPY --from=builder --chown=harmonyuser:harmonygroup /app/migrate .
COPY --chown=harmonyuser:harmonygroup config/ config/
COPY --chown=harmonyuser:harmonygroup migrations/ migrations/
COPY --chown=harmonyuser:harmonygroup public/ public/
COPY --chown=harmonyuser:harmonygroup templates/ templates/
COPY --chown=harmonyuser:harmonygroup translations/ translations/
COPY --chown=harmonyuser:harmonygroup docs/templates/paris docs/templates/paris

USER harmonyuser
ENV PORT=8213
EXPOSE 8213

ENTRYPOINT ["./web"]