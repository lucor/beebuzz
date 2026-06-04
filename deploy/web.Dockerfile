# Stage 1: Build web
FROM node:24-alpine AS web-builder
WORKDIR /build/web

RUN corepack enable

COPY web/package.json web/pnpm-lock.yaml web/pnpm-workspace.yaml ./
COPY web/apps/site/package.json ./apps/site/
COPY web/apps/hive/package.json ./apps/hive/
COPY web/packages/shared/package.json ./packages/shared/

RUN pnpm install --frozen-lockfile

COPY web/ ./
COPY docs/openapi.yaml /build/docs/openapi.yaml

ARG VITE_BEEBUZZ_DOMAIN
ARG VITE_BEEBUZZ_DEBUG=false
ARG VITE_BEEBUZZ_DEPLOYMENT_MODE=self_hosted

ENV VITE_BEEBUZZ_DOMAIN=$VITE_BEEBUZZ_DOMAIN \
    VITE_BEEBUZZ_DEBUG=$VITE_BEEBUZZ_DEBUG \
    VITE_BEEBUZZ_DEPLOYMENT_MODE=$VITE_BEEBUZZ_DEPLOYMENT_MODE

RUN pnpm run build

# Stage 2: Caddy + static files
FROM caddy:2.11-alpine

ARG VERSION=dev
ARG SOURCE_REPO=https://codeberg.org/beebuzz/beebuzz

LABEL org.opencontainers.image.title="BeeBuzz Web" \
      org.opencontainers.image.description="BeeBuzz web frontend" \
      org.opencontainers.image.url="https://beebuzz.app" \
      org.opencontainers.image.documentation="https://beebuzz.app/docs" \
      org.opencontainers.image.vendor="BeeBuzz" \
      org.opencontainers.image.licenses="AGPL-3.0-only" \
      org.opencontainers.image.version=${VERSION} \
      org.opencontainers.image.source=${SOURCE_REPO}

COPY --from=web-builder /build/web/build /srv/build
COPY deploy/Caddyfile /etc/caddy/Caddyfile

# BEEBUZZ_DOMAIN is needed at runtime by Caddy for {$BEEBUZZ_DOMAIN} expansion.
# It carries the same value as VITE_BEEBUZZ_DOMAIN used at build time.
ARG VITE_BEEBUZZ_DOMAIN
ENV BEEBUZZ_DOMAIN=${VITE_BEEBUZZ_DOMAIN}

EXPOSE 80
