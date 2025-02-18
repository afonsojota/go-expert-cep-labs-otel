FROM golang:1.21

WORKDIR /app
COPY . .

RUN go build -o service-a .

CMD ["./service-a"]