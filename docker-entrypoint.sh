#!/bin/sh
set -e

APP_USER="user"
APP_UID="1000"
SOCKET="/var/run/docker.sock"

# When the Docker socket is bind-mounted in, its group ownership reflects
# whatever GID the docker group happens to have on the host, which varies
# from machine to machine and isn't known at image build time. Join that
# group here, at container start, instead of baking in a fixed GID.
if [ -S "$SOCKET" ]; then
	socket_gid="$(stat -c '%g' "$SOCKET")"
	socket_group="$(getent group "$socket_gid" | cut -d: -f1)"

	if [ -z "$socket_group" ]; then
		socket_group="docker-sock"
		groupadd -g "$socket_gid" "$socket_group"
	fi

	usermod -aG "$socket_group" "$APP_USER"
fi

export HOME="/home/$APP_USER"
exec setpriv --reuid="$APP_UID" --regid="$APP_UID" --init-groups "$@"
