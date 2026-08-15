FROM golang:1.26 AS build-stage

WORKDIR /app

COPY go.mod go.sum .
RUN go mod download

COPY . .

RUN apt-get update && \
    apt-get upgrade -y && \
    apt-get install -y build-essential make

RUN make build

# Static docker CLI + compose plugin, without pulling in a full Docker
# Engine/containerd install or a third-party apt repo just for the CLI.
FROM docker:27-cli AS docker-cli-stage

FROM debian:12-slim AS build-release-stage

RUN apt-get update && \
    apt-get upgrade -y && \
    apt-get install -y git && \
    rm -rf /var/lib/apt/lists/*

COPY --from=docker-cli-stage /usr/local/bin/docker /usr/local/bin/docker
COPY --from=docker-cli-stage /usr/local/libexec/docker/cli-plugins/docker-compose /usr/local/libexec/docker/cli-plugins/docker-compose

# Create a system group and user named "user" to run upstream-watch as.
# The entrypoint grants this user access to the mounted docker.sock at
# container start (see docker-entrypoint.sh) and then drops root.
RUN groupadd -g 1000 -r user && \
    useradd -r -m -u 1000 -g 1000 user

WORKDIR /workdir
RUN chown -R user:user /workdir

COPY --from=build-stage /app/upstream-watch /app/upstream-watch
COPY docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh

ENTRYPOINT ["docker-entrypoint.sh"]
CMD ["/app/upstream-watch", "/workdir"]
