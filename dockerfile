FROM golang:1.23-alpine

WORKDIR /app

COPY . .

RUN go mod tidy

RUN go build -o /runner ./cmd/runner/main.go

CMD ["/runner"]