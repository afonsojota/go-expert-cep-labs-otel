FROM golang:1.21

WORKDIR /app
COPY . .

RUN go mod tidy
RUN go build -o service-b .

CMD ["./service-b"]