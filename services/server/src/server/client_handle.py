import socket
import logger
import threading
from lottery.lottery import Lottery, Bet
from .server_protocol import ServerProtocol
from .rw_lock import ReadWriteLock


class ClientHandle(threading.Thread):
    def __init__(
        self,
        client_socket: socket.socket,
        storage_path: str,
        agency_quorum_min: int,
        agency_counter: list,
        agency_counter_lock: threading.Lock,
        lottery_ready_event: threading.Event,
        rw_lock: ReadWriteLock,
    ) -> None:
        super().__init__()
        self.client_socket = client_socket
        self.lottery = Lottery(storage_path)
        self.protocol = ServerProtocol()
        self.agency_quorum_min = agency_quorum_min
        self.agency_counter = agency_counter        
        self.agency_counter_lock = agency_counter_lock
        self.lottery_ready_event = lottery_ready_event
        self.rw_lock = rw_lock

    def run(self) -> None:
        action = "handle-client"
        agency_id = None
        total_bets_received = 0
        try:
            logger.info(action, logger.LogResult.in_progress)

            while True:
                batch_reciv = self.protocol.receive_batch(self.client_socket)
                if batch_reciv is None:
                    break

                if batch_reciv and agency_id is None:
                    agency_id = batch_reciv[0].agency_id

                total_bets_received += len(batch_reciv)

                self.rw_lock.acquire_write()
                try:
                    self.lottery.store_bets(batch_reciv)
                finally:
                    self.rw_lock.release_write()

            # aumentamos el contador de agencias que terminaron y verificamos si se alcanza el quórum
            with self.agency_counter_lock:
                self.agency_counter[0] += 1
                if self.agency_counter[0] >= self.agency_quorum_min:
                    self.lottery_ready_event.set()  

            # si no hay quórum, esperamos a que se alcance antes de procesar las apuestas
            self.lottery_ready_event.wait()

            self.rw_lock.acquire_read()
            try:
                all_bets = self.lottery.load_bets()
            finally:
                self.rw_lock.release_read()

            agency_winners = [
                b for b in all_bets
                if b.agency_id == agency_id and self.lottery.has_won(b)
            ]

            self.protocol.send_winners(self.client_socket, agency_winners)

            logger.info(
                action,
                logger.LogResult.success,
                "bets-processed",
                total_bets_received,
            )
        except Exception as e:
            logger.error(action, logger.LogResult.fail, "error", str(e))
            raise e
        finally:
            self.client_socket.close()