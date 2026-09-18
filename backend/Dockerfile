# syntax=docker/dockerfile:1
FROM golang:1.24-alpine AS build

RUN apk add --no-cache ca-certificates git
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/similar-items ./cmd/similar

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata wget \
    && addgroup -S app \
    && adduser -S -G app app
WORKDIR /app
COPY --from=build /out/similar-items ./similar-items
USER app
EXPOSE 8080
ENTRYPOINT ["./similar-items"]
