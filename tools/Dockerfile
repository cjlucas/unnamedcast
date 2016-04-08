FROM golang:1
MAINTAINER Chris Lucas

ENV GOPATH /app
ENV PATH $PATH:/app/bin

COPY . /app/
WORKDIR /app/
RUN make fix
