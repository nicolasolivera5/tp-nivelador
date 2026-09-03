import socket
import logger
import threading
from lottery.lottery import Lottery, Bet
from .server_protocol import ServerProtocol


class ClientHandle(threading.Thread):
    def __init__(self, client_socket: socket.socket, storage_path: str,barrier_agency_quorum: threading.Barrier, lock_lottery: threading.Lock) -> None:
        super().__init__()
        self.client_socket = client_socket
        self.lottery = Lottery(storage_path)
        self.protocol = ServerProtocol()
        self.barrier_agency_quorum = barrier_agency_quorum
        self.lock_lottery = lock_lottery
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

                with self.lock_lottery:
                    self.lottery.store_bets(batch_reciv)

            self.barrier_agency_quorum.wait()

            with self.lock_lottery:
                all_bets = self.lottery.load_bets()

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