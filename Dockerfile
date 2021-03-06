FROM golang:1.9.2-alpine3.6

RUN apk update && apk upgrade && \
    apk add --no-cache git bash



WORKDIR /go/src/github.com/ear7h/edns/child
COPY . .

RUN go get -v ./...
RUN go build

CMD ["./child"]