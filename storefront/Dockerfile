FROM node:24-alpine AS build
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
RUN npm run build

# Runtime: Node runs server/*.ts directly (type stripping); only the
# SSR bundle's runtime dependencies are installed.
FROM node:24-alpine
WORKDIR /app
ENV NODE_ENV=production PORT=3100
COPY package.json package-lock.json ./
RUN npm ci --omit=dev && npm cache clean --force
COPY --from=build /app/dist ./dist
COPY server ./server
USER node
EXPOSE 3100
HEALTHCHECK --interval=30s --timeout=3s CMD wget -qO- http://127.0.0.1:3100/healthz || exit 1
CMD ["node", "server/main.ts"]
