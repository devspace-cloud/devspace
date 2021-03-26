FROM golang:1.13.0-alpine

ADD . /go/src/app
WORKDIR /go/src/app

RUN cd /go/src/app && go get ./... && go build . && chmod +x /go/src/app/app

CMD ["/go/src/app/app"]
