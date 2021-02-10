FROM golang:1.15.8

RUN mkdir /app

ADD . /app

WORKDIR /app

RUN go build -o watch cmd/main.go

CMD ["/app/watch"]
