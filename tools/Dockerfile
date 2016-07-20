FROM golang:1.6-alpine
MAINTAINER Chris Lucas

RUN apk update
RUN apk add make nodejs

ENV PATH $PATH:/app/bin
RUN mkdir -p /app/bin
RUN mkdir -p /app/dashboard/dist

COPY . /build
WORKDIR /build/dashboard
RUN npm install && node_modules/.bin/webpack && cp -rv dist/* /app/dashboard/dist

ENV GOPATH /build
WORKDIR /build
RUN make install && cp bin/* /app/bin

WORKDIR /app
RUN rm -rf /build
