FROM node:22-alpine AS frontend
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY index.html vite.config.ts ./
COPY src ./src
RUN npm run build

FROM golang:1.26-alpine AS backend
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/url-shortener ./cmd/server

FROM alpine:3.22 AS runtime
RUN addgroup -S -g 10001 app && adduser -S -D -H -u 10001 -G app app
WORKDIR /app
COPY --from=backend --chown=10001:10001 /out/url-shortener ./url-shortener
COPY --from=backend --chown=10001:10001 /src/migrations ./migrations
COPY --from=frontend --chown=10001:10001 /app/web/dist ./web/dist
USER 10001:10001
EXPOSE 8080
ENTRYPOINT ["./url-shortener"]
