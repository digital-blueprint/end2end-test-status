# ---- Build frontend ----
FROM node:24-alpine AS frontend-builder
WORKDIR /app/frontend
COPY frontend/package.json frontend/package-lock.json* ./
RUN npm ci
COPY frontend/ ./
ARG PATH_PREFIX
ENV PATH_PREFIX=${PATH_PREFIX}
ARG VITE_FOOTER_TEXT
ENV VITE_FOOTER_TEXT=${VITE_FOOTER_TEXT}
RUN npm run build

# ---- Build Go binary ----
FROM golang:1.25-alpine AS go-builder
# CGO is required for go-sqlite3
RUN apk add --no-cache gcc musl-dev
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
# Copy built frontend assets so they can be embedded
COPY --from=frontend-builder /app/frontend/dist ./frontend/dist
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-w -s" -o e2e-status .

# ---- Final image ----
FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=go-builder /app/e2e-status .

VOLUME ["/data"]
ENV PORT=8080
EXPOSE 8080

ENTRYPOINT ["/app/e2e-status"]
