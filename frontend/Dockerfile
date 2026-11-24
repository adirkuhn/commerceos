# The build output is static files, so the builder runs on the build
# machine's own platform (no emulation); only the nginx runtime is per-target.
FROM --platform=$BUILDPLATFORM node:20-alpine AS builder
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci --no-audit --no-fund
COPY . .
ARG VITE_LM2_LOGIN_URL
ARG VITE_LM2_CALLBACK_PATH
ARG VITE_API_BASE_URL
ARG VITE_STOREFRONT_URL
RUN npm run build

FROM nginx:1.27-alpine
COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
# Writes config.js from the server's .env at every start (the nginx image runs
# /docker-entrypoint.d/*.sh before nginx). The VITE_* build args above are
# only fallbacks for local builds.
COPY --chmod=755 docker/write-config.sh /docker-entrypoint.d/40-commerceos-config.sh
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
