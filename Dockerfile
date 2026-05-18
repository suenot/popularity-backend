# syntax=docker/dockerfile:1
FROM golang:1.25-alpine AS build
WORKDIR /src
RUN apk add --no-cache git ca-certificates
ENV CGO_ENABLED=0 GOOS=linux

# Cache module downloads.
COPY go.mod go.sum ./
RUN go mod download

# Build both binaries.
COPY . .
RUN go build -trimpath -ldflags='-s -w' -o /out/api ./cmd/api \
 && go build -trimpath -ldflags='-s -w' -o /out/scheduler ./cmd/scheduler

FROM alpine:3.20
# yt-dlp is required by the YouTube parser (no API key path). It pulls in
# python3 + ffmpeg for video URL/metadata extraction.
RUN apk add --no-cache ca-certificates tzdata python3 py3-pip yt-dlp \
 && adduser -D -u 10001 app
USER app
WORKDIR /app
COPY --from=build /out/api /app/api
COPY --from=build /out/scheduler /app/scheduler

ENV BACKEND_PORT=8080
EXPOSE 8080
ENTRYPOINT ["/app/api"]
