import socket
import logger
import threading
import os
import signal
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
        
        self.running = True
        self.server_socket = None
        self.client_threads = []

        signal.signal(signal.SIGTERM, self._handle_signal)
        #signal.signal(signal.SIGINT, self._handle_signal)

    def _handle_signal(self, signum, frame):
        logger.info("signal-handler", logger.LogResult.in_progress, "signal", signum)
        self.running = False
        
        try:
            self.barrier_agency_quorum.abort()
        except Exception:
            pass

        if self.server_socket:
            try:
                self.server_socket.shutdown(socket.SHUT_RDWR)
            except Exception:
                pass
            self.server_socket.close()

    def run(self):
        action = "accept-connection"
        self.server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self.server_socket.bind((self.server_host, self.server_port))
        self.server_socket.listen()

        while self.running:
            try:
                logger.info(action, logger.LogResult.in_progress)
                client_socket, _ = self.server_socket.accept()
            except OSError:
                break
            except Exception as e:
                if not self.running:
                    break
                logger.error(action, logger.LogResult.fail)
                raise e

            logger.info(action, logger.LogResult.success)
            
            client_handle = ClientHandle(
                client_socket, 
                self.storage_path, 
                self.barrier_agency_quorum, 
                self.lock_lottery
            )
            client_handle.start()
            self.client_threads.append(client_handle)

        for t in self.client_threads:
            t.join()

        logger.info("server-shutdown", logger.LogResult.success)