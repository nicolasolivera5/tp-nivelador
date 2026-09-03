import socket
import logger
import threading
import os
from  .clientHandle import ClientHandle

STORAGE_FILE = os.path.normpath(os.path.join(os.path.dirname(os.path.abspath(__file__)), "storage", "bets.csv"))
os.makedirs(os.path.dirname(STORAGE_FILE), exist_ok=True)

class Server:
    def __init__(self, server_host: str, server_port: int) -> None:
        self.server_host = server_host
        self.server_port = server_port
        self.storage_path = STORAGE_FILE
        self.barrier_agency_quorum = threading.Barrier(int(os.environ["AGENCY_QUORUM_MIN"]))
        self.lock_lottery = threading.Lock()

    def run(self):
        action = "accept-connection"
        with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as server_socket:
            server_socket.bind((self.server_host, self.server_port))
            server_socket.listen()
            while True:
                try:
                    logger.info(action, logger.LogResult.in_progress)
                    client_socket, _ = server_socket.accept()
                except Exception as e:
                    logger.error(action, logger.LogResult.fail)
                    raise e
                logger.info(action, logger.LogResult.success)
                client_handle = ClientHandle(client_socket, self.storage_path, self.barrier_agency_quorum, self.lock_lottery)
                client_handle.start()