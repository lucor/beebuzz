# Stage 1: Build server
FROM golang:1.26.4-alpine AS server-builder
WORKDIR /build

ARG COMMIT_SHA=dev

COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN CGO_ENABLED=0 go build -trimpath \
    -ldflags="-w -s -X main.commitHash=$(echo ${COMMIT_SHA} | cut -c1-7)" \
    -o beebuzzd .

# Stage 2: Runtime image
FROM gcr.io/distroless/base-debian12
WORKDIR /app

ARG VERSION=dev
ARG SOURCE_REPO=https://codeberg.org/beebuzz/beebuzz

LABEL org.opencontainers.image.title="BeeBuzz Server" \
      org.opencontainers.image.description="BeeBuzz server — web push notification delivery" \
      org.opencontainers.image.url="https://beebuzz.app" \
      org.opencontainers.image.documentation="https://docs.beebuzz.app" \
      org.opencontainers.image.vendor="BeeBuzz" \
      org.opencontainers.image.licenses="AGPL-3.0-only" \
      org.opencontainers.image.version=${VERSION} \
      org.opencontainers.image.source=${SOURCE_REPO}

COPY --from=server-builder /build/beebuzzd /app/beebuzzd

VOLUME ["/var/lib/beebuzz/db", "/var/lib/beebuzz/attachments"]

ENV BEEBUZZ_ENV=production
ENV BEEBUZZ_DB_DIR=/var/lib/beebuzz/db
ENV BEEBUZZ_ATTACHMENTS_DIR=/var/lib/beebuzz/attachments
ENV BEEBUZZ_PORT=8899

EXPOSE 8899

HEALTHCHECK --interval=30s --timeout=5s --start-period=30s --retries=3 \
	CMD ["/app/beebuzzd", "healthcheck"]

ENTRYPOINT ["/app/beebuzzd"]
CMD ["serve"]
