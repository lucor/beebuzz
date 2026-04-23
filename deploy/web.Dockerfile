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

ARG VITE_BEEBUZZ_DOMAIN
ARG VITE_BEEBUZZ_DEBUG=false
ARG VITE_BEEBUZZ_DEPLOYMENT_MODE=self_hosted

ENV VITE_BEEBUZZ_DOMAIN=$VITE_BEEBUZZ_DOMAIN \
    VITE_BEEBUZZ_DEBUG=$VITE_BEEBUZZ_DEBUG \
    VITE_BEEBUZZ_DEPLOYMENT_MODE=$VITE_BEEBUZZ_DEPLOYMENT_MODE

RUN pnpm run build

# Stage 2: Caddy + static files
FROM caddy:alpine
COPY --from=web-builder /build/web/build /srv/build
COPY deploy/Caddyfile /etc/caddy/Caddyfile
# BEEBUZZ_DOMAIN is needed at runtime by Caddy for {$BEEBUZZ_DOMAIN} expansion.
# It carries the same value as VITE_BEEBUZZ_DOMAIN used at build time.
ARG VITE_BEEBUZZ_DOMAIN
ENV BEEBUZZ_DOMAIN=${VITE_BEEBUZZ_DOMAIN}
EXPOSE 80
