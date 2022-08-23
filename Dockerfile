FROM golang:1.19-alpine as builder

WORKDIR /app

COPY go.mod ./
COPY go.sum ./

RUN go mod download

COPY RESTfullApi/ RESTfullApi/ 

RUN cd RESTfullApi/ && go build  -ldflags="-w -s" -o main

FROM alpine

WORKDIR /app

# COPY --from=builder RESTfullApi/main main
COPY --from=builder /app/RESTfullApi/main /app/main

# USER nonroot:nonroot

# ENTRYPOINT ["/main"]
