# Build and dependency stages run on the build machine's own platform (no
# emulation): the runtime dependencies are plain JavaScript, so the same
# node_modules works on every target. Only the final Node image is per-target.
FROM --platform=$BUILDPLATFORM node:24-alpine AS build
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM --platform=$BUILDPLATFORM node:24-alpine AS deps
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci --omit=dev && npm cache clean --force

# Runtime: Node runs server/*.ts directly (type stripping); only the
# SSR bundle's runtime dependencies are installed.
FROM node:24-alpine
WORKDIR /app
ENV NODE_ENV=production PORT=3100
COPY package.json package-lock.json ./
COPY --from=deps /app/node_modules ./node_modules
COPY --from=build /app/dist ./dist
COPY server ./server
USER node
EXPOSE 3100
HEALTHCHECK --interval=30s --timeout=3s CMD wget -qO- http://127.0.0.1:3100/healthz || exit 1
CMD ["node", "server/main.ts"]
