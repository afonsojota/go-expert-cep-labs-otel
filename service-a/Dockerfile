FROM golang:1.21

WORKDIR /app
COPY . .

RUN go mod init service-a

RUN go mod tidy

RUN go build -o service-a .

CMD ["./service-a"]