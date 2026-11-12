#!/usr/bin/env bash

set -eo pipefail

# Fix volume perms (old deployments are root-owned) then drop to mesh.
# tcpdump gets NET_RAW from its file capability, so nothing is kept ambient.
if [ "$(id -u)" = "0" ]; then
    if ! su -s /bin/sh mesh -c 'test -w /home/mesh/acquisitions'; then
        chown mesh:mesh /home/mesh/acquisitions ||
            echo "warning: cannot chown /home/mesh/acquisitions; acquisitions may not be writable" >&2
    fi
    export HOME=/home/mesh USER=mesh
    exec setpriv \
        --reuid=mesh --regid=mesh --init-groups \
        -- "$0" "$@"
fi

exec "$@"
