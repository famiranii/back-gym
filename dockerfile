FROM golang:1.26-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum ./

ENV GOPROXY=direct

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o server .


FROM alpine:3.22

WORKDIR /app

RUN apk add --no-cache ca-certificates

COPY --from=builder /app/server ./server

COPY --from=builder /app/internal/db/migration ./internal/db/migration

COPY app.docker.env ./app.env

RUN mkdir -p /app/uploads

EXPOSE 8080

CMD ["./server"]