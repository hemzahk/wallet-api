FROM golang:1.25.5-alpine AS builder

WORKDIR /app

COPY go.mod go.sum* ./

RUN go mod download

COPY . .

RUN go build -o main cmd/api/*.go

FROM alpine:3.23

WORKDIR /app

COPY --from=builder /app/main .

EXPOSE 3000

CMD ["./main"]