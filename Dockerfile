FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o server ./cmd/server/

FROM alpine:3.22

WORKDIR /app

COPY --from=builder  /app/server /app/
COPY --from=builder /app/config/ ./config

CMD [ "./server" ]