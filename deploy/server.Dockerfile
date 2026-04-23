# Stage 1: Build server
FROM golang:1.26-alpine AS server-builder
WORKDIR /build

ARG COMMIT_SHA=dev

COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN CGO_ENABLED=0 go build -trimpath \
    -ldflags="-w -s -X main.commitHash=$(echo ${COMMIT_SHA} | cut -c1-7)" \
    -o beebuzz-server ./cmd/beebuzz-server

# Stage 2: Runtime image
FROM gcr.io/distroless/base-debian12
WORKDIR /app

COPY --from=server-builder /build/beebuzz-server /app/beebuzz-server

VOLUME ["/var/lib/beebuzz/db", "/var/lib/beebuzz/attachments"]

ENV BEEBUZZ_ENV=production
ENV BEEBUZZ_DB_DIR=/var/lib/beebuzz/db
ENV BEEBUZZ_ATTACHMENTS_DIR=/var/lib/beebuzz/attachments
ENV BEEBUZZ_PORT=8899

EXPOSE 8899

HEALTHCHECK --interval=30s --timeout=5s --start-period=30s --retries=3 \
	CMD ["/app/beebuzz-server", "healthcheck"]

ENTRYPOINT ["/app/beebuzz-server"]
CMD ["serve"]
