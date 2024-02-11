FROM golang:latest

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build main.go

RUN find . -mindepth 1 ! -name main ! -path './scripts*' -exec rm -rf {} +

CMD rm -rf /mnt/scripts/* && cp -r scripts/* /mnt/scripts && /app/main
