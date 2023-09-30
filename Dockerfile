FROM golang:1.21-alpine as builder

WORKDIR /app

COPY go.mod ./
COPY go.sum ./

RUN go mod download

COPY RESTfullApi/ RESTfullApi/ 

# We can further reduce the size with -ldflags= "-s -w", removing the DWARF
# and symbols tables with other debug information. I would not recommend the
# latter option, as non-DWARF elements allow important runtime routines, like
# gathering profiles
RUN cd RESTfullApi/ && go build  -ldflags="-w" -o main

FROM alpine

WORKDIR /app

# COPY --from=builder RESTfullApi/main main
COPY --from=builder /app/RESTfullApi/main /app/main

# USER nonroot:nonroot

# ENTRYPOINT ["/main"]
