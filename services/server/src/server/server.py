import socket
import logger
from lottery.lottery import Lottery, Bet
from .server_protocol import ServerProtocol


class Server:
    def __init__(self, server_host: str, server_port: int, storage_path: str) -> None:
        self.server_host = server_host
        self.server_port = server_port
        self.lottery = Lottery(storage_path)
        self.protocol = ServerProtocol()

    def _handle_client(self, client_socket):
        action = "handle-client"
        received_bets: list[Bet] = []
        try:
            logger.info(action, logger.LogResult.in_progress)
            while True:
                         
                bet_reciv = self.protocol.receive_bet(client_socket)
                if bet_reciv is None:
                    break
                received_bets.append(bet_reciv)

            self.lottery.store_bets(received_bets)
            
            agency_winners = [
                b for b in self.lottery.load_bets() 
                if self.lottery.has_won(b)
            ]
            
            self.protocol.send_winners(client_socket, agency_winners)

            logger.info(
                action,
                logger.LogResult.success,
                "bets-processed",
                len(received_bets),
            )

        except Exception as e:
            logger.error(action, logger.LogResult.fail, "error", str(e))
            raise e

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

                self._handle_client(client_socket)
