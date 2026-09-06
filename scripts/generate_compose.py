#!/usr/bin/env python3
"""Generate a Docker Compose file with a configurable number of clients."""

import argparse
from pathlib import Path


def client_service(client_id: int) -> str:
    return f"""  client_{client_id}:
    build:
      context: ./services/client
      dockerfile: Dockerfile
    container_name: client_{client_id}
    depends_on:
      - server
    environment:
      - AGENCY_ID={client_id}
      - SERVER_HOST=server
      - SERVER_PORT=5678
      - INPUT_FILE=/input/input-{client_id}.csv
      - OUTPUT_FILE=/output/output-{client_id}.csv
      - BATCH_SIZE=40
    volumes:
      - ./input:/input:ro
      - ./output:/output
"""


def compose_file(client_count: int) -> str:
    clients = "\n".join(client_service(client_id) for client_id in range(client_count))
    return f"""services:
  server:
    build:
      context: ./services/server
      dockerfile: Dockerfile
    container_name: server
    environment:
      - PYTHONUNBUFFERED=1
      - SERVER_HOST=server
      - SERVER_PORT=5678
      - AGENCY_QUORUM_MIN={client_count}
    ports:
      - \"5678:5678\"

{clients}"""


def main() -> None:
    parser = argparse.ArgumentParser(
        description="Generate a Docker Compose file with N lottery clients."
    )
    parser.add_argument("client_count", type=int, help="number of clients to configure")
    parser.add_argument(
        "output",
        nargs="?",
        type=Path,
        default=Path("docker-compose.generated.yaml"),
        help="destination file (default: docker-compose.generated.yaml)",
    )
    args = parser.parse_args()

    if args.client_count < 1:
        parser.error("client_count must be greater than zero")

    args.output.write_text(compose_file(args.client_count), encoding="utf-8")
    print(f"Generated {args.output} with {args.client_count} clients")


if __name__ == "__main__":
    main()