FROM golang:1.21 AS build-stage

WORKDIR /app

COPY go.mod go.sum .
RUN go mod download

COPY . .

RUN apt-get update && \
    apt-get upgrade -y

RUN apt-get install -y build-essential make

RUN make build

FROM gcr.io/distroless/base-debian12 AS build-release-stage

WORKDIR /

COPY --from=build-stage /app/upstream-watch /app/upstream-watch

USER nonroot:nonroot

ENTRYPOINT ["/app/upstream-watch", "/workdir"]