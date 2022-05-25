# syntax=docker/dockerfile:1

FROM golang:1.18-alpine as builder

WORKDIR /app

COPY go.mod ./
COPY go.sum ./

RUN go mod download

COPY RESTfullApi/ RESTfullApi/ 

RUN cd RESTfullApi/ && go build  -ldflags="-w -s" -o main

FROM alpine

WORKDIR /

# COPY --from=builder RESTfullApi/main main
COPY --from=builder app/RESTfullApi/main main

# USER nonroot:nonroot

# ENTRYPOINT ["/main"]
