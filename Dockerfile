FROM golang:1.23-alpine

WORKDIR /app

COPY . .

RUN go build -o bin/telemetry server.go

EXPOSE 1323

CMD ["./bin/telemetry"]