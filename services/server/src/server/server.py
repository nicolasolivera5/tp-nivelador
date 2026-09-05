import socket
import logger
import threading
import os
import signal
from  .client_handle import ClientHandle, ReadWriteLock

STORAGE_FILE = os.path.normpath(os.path.join(os.path.dirname(os.path.abspath(__file__)), "storage", "bets.csv"))
os.makedirs(os.path.dirname(STORAGE_FILE), exist_ok=True)

class Server:
    def __init__(self, server_host: str, server_port: int) -> None:
        self.server_host = server_host
        self.server_port = server_port
        self.storage_path = STORAGE_FILE
        self.agency_quorum_min = int(os.environ["AGENCY_QUORUM_MIN"])
        self.agency_counter = [0]                   
        self.agency_counter_lock = threading.Lock() 
        self.lottery_ready_event = threading.Event() 
        self.rw_lock = ReadWriteLock()              
        
        self.running = True
        self.server_socket = None
        self.client_threads = []

        signal.signal(signal.SIGTERM, self._handle_signal)
    
    def _handle_signal(self, signum, frame):
        logger.info("signal-handler", logger.LogResult.in_progress, "signal", signum)
        self.running = False
        
        self.lottery_ready_event.set()

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
                self.agency_quorum_min,
                self.agency_counter,
                self.agency_counter_lock,
                self.lottery_ready_event,
                self.rw_lock,
            )
            client_handle.start()
            self.client_threads.append(client_handle)

        for t in self.client_threads:
            t.join()

        logger.info("server-shutdown", logger.LogResult.success)