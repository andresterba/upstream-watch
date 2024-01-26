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

RUN apt-get install -y \
    git \
    docker \
    docker-compose

# Create a system group named "user" with the -r flag
RUN groupadd -g 1000 -r user

# Create a system user named "user" and add it to the "user" group with the -r and -g flags
RUN useradd -r -u 1000 -g 1000 user

RUN usermod -aG docker user

WORKDIR /workdir

# Change the ownership of the working directory to the non-root user "user"

RUN chown -R user:user /workdir

# Switch to the non-root user "user"
USER user


COPY --from=build-stage /app/upstream-watch /app/upstream-watch

CMD ["/app/upstream-watch", "/workdir"]