FROM golang:1.25-alpine AS builder

WORKDIR /app

RUN apk add --no-cache ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -o /bin/api ./cmd/api

FROM alpine:3.20

RUN addgroup -S app && adduser -S app -G app
USER app

WORKDIR /home/app
COPY --from=builder /bin/api /home/app/api
COPY --from=builder /app/static /home/app/static

EXPOSE 8080
ENTRYPOINT ["/home/app/api"]

