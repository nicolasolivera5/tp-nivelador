#!/usr/bin/env bash
set -euo pipefail

server_container="${1:-server}"
server_port="${SERVER_PORT:-5678}"

if ! docker inspect "$server_container" >/dev/null 2>&1; then
    printf 'Container not found: %s\n' "$server_container" >&2
    exit 1
fi

mapfile -t networks < <(
    docker inspect \
        --format '{{range $network, $_ := .NetworkSettings.Networks}}{{$network}}{{"\n"}}{{end}}' \
        "$server_container"
)

if [[ ${#networks[@]} -eq 0 || -z "${networks[0]}" ]]; then
    printf 'No Docker network found for container: %s\n' "$server_container" >&2
    exit 1
fi

network="${networks[0]}"
printf 'Checking %s:%s from Docker network %s\n' "$server_container" "$server_port" "$network"

docker run --rm --network "$network" alpine:3.20 \
    sh -c 'nc -z -w 3 "$1" "$2"' sh "$server_container" "$server_port"

printf 'Server is accepting TCP connections\n'