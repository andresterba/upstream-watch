FROM golang:1.21 AS build-stage

WORKDIR /app

COPY go.mod go.sum .
RUN go mod download

COPY . .

RUN apt-get update && \
    apt-get upgrade -y

RUN apt-get install -y build-essential make

RUN make build

FROM debian:12-slim AS build-release-stage

RUN apt-get update && \
    apt-get upgrade -y

RUN apt-get install -y git

WORKDIR /

COPY --from=build-stage /app/upstream-watch /app/upstream-watch

CMD ["/app/upstream-watch", "/workdir"]