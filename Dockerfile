FROM node:22-alpine AS frontend
WORKDIR /build/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN HOME=/tmp npm ci
COPY frontend/ .
RUN HOME=/tmp npm run build

FROM golang:1.26-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /build/frontend/build/ frontend/build/
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o waypoint .

FROM alpine:3.22
RUN apk add --no-cache ca-certificates
COPY --from=builder /build/waypoint /usr/local/bin/waypoint
EXPOSE 7666 7667
ENTRYPOINT ["waypoint"]
