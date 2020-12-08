#!/usr/bin/dumb-init /bin/sh
set -e

su-exec revproxy:revporxy /app/revproxy "$@"
