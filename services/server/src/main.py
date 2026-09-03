import os
import sys

import logger
import server

SERVER_HOST = os.environ["SERVER_HOST"]
SERVER_PORT = int(os.environ["SERVER_PORT"])
STORAGE_FILE = os.path.normpath(os.path.join(os.path.dirname(os.path.abspath(__file__)), "storage", "bets.csv"))
os.makedirs(os.path.dirname(STORAGE_FILE), exist_ok=True)

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
