FROM golang:latest

COPY source /go/src

ENV GOPATH=

WORKDIR /go/src/microservice-framework
RUN go mod init github.com/Dartmouth-OpenAV/microservice-framework
RUN go mod tidy

WORKDIR /go
RUN go mod init github.com/Dartmouth-OpenAV/microservice-roku-fpd
RUN go mod edit -replace github.com/Dartmouth-OpenAV/microservice-framework=./src/microservice-framework
RUN go mod tidy

WORKDIR /go/src
RUN go get -u
RUN go build -o /go/bin/microservice

ENTRYPOINT /go/bin/microservice
