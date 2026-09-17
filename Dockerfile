FROM node:24-alpine AS frontend-build
WORKDIR /src/Frontend
COPY Frontend/package.json Frontend/package-lock.json ./
RUN npm ci
COPY Frontend/ ./
RUN npm run build

FROM golang:1.26-alpine AS backend-build
WORKDIR /src/Backend
COPY Backend/go.mod ./
RUN go mod download
COPY Backend/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server .

FROM alpine:3.24 AS runtime
RUN addgroup -S app && adduser -S -G app -u 10001 app
WORKDIR /app
COPY --from=backend-build /out/server /app/server
COPY --from=frontend-build /src/Frontend/dist /app/static
ENV PORT=8080 STATIC_DIR=/app/static
USER app:app
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -q -O /dev/null "http://127.0.0.1:${PORT}/api/health" || exit 1
ENTRYPOINT ["/app/server"]
