import os
import sys

import logger
import server
from pathlib import Path

SERVER_HOST = os.environ["SERVER_HOST"]
SERVER_PORT = int(os.environ["SERVER_PORT"])
STORAGE_FILE = Path(__file__).resolve().parent / "server" / "storage" / "bets.csv"

STORAGE_FILE.parent.mkdir(parents=True, exist_ok=True)
STORAGE_FILE.touch(exist_ok=True)

def main():
    logger.init()
    s = server.Server(SERVER_HOST, SERVER_PORT, str(STORAGE_FILE))
    try:
        s.run()
    except Exception as e:
        logger.error("server-run", logger.LogResult.fail, "err", e)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
