FROM golang:1-alpine
MAINTAINER Chris Lucas

RUN apk update
RUN apk add make

ENV GOPATH /app
ENV PATH $PATH:/app/bin

COPY . /app/
WORKDIR /app/
RUN make install
