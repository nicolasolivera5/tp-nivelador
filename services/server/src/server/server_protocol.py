import logger
import socket
import safe_socket
from lottery.lottery import Bet

_AGENCY_ID_SIZE = 2
_BATCH_COUNT_SIZE = 2
_NAME_LENGTH_SIZE = 1
_LAST_NAME_LENGTH_SIZE = 1
_DOCUMENT_SIZE = 4
_BIRTHDATE_SIZE = 10
_NUMBER_SIZE = 4

class ServerProtocol:
    def receive_batch(self, client_socket: socket.socket) -> list[Bet] | None:
        action = "receive-batch"
        try:
            # encabezado del batch (4 bytes)
            header_bytes = safe_socket.recv_all(client_socket, _AGENCY_ID_SIZE + _BATCH_COUNT_SIZE)
            if not header_bytes:
                return None
            
            agency_id = int.from_bytes(header_bytes[:2], byteorder="big")
            count = int.from_bytes(header_bytes[2:4], byteorder="big")

            # fin de envios de esta agencia
            if count == 0:
                # mandamos ACK de fin y salir
                safe_socket.send_all(client_socket, b"\x00")
                logger.info("receive-end", logger.LogResult.success, "agency_id", agency_id)
                return None

            bets = []
            for _ in range(count):  

                # tamanio del nombre (1 byte)
                name_length_bytes = safe_socket.recv_all(client_socket, _NAME_LENGTH_SIZE)
                name_length = int.from_bytes(name_length_bytes, byteorder="big")
                name_bytes = safe_socket.recv_all(client_socket, name_length)

                last_name_length_bytes = safe_socket.recv_all(client_socket, _LAST_NAME_LENGTH_SIZE)
                last_name_length = int.from_bytes(last_name_length_bytes, byteorder="big")
                last_name_bytes = safe_socket.recv_all(client_socket, last_name_length)

                document_bytes = safe_socket.recv_all(client_socket, _DOCUMENT_SIZE)
                birthdate_bytes = safe_socket.recv_all(client_socket, _BIRTHDATE_SIZE)
                number_bytes = safe_socket.recv_all(client_socket, _NUMBER_SIZE)

                bet = Bet(
                    agency_id=agency_id,
                    first_name=name_bytes.decode("utf-8"),
                    last_name=last_name_bytes.decode("utf-8"),
                    document=int.from_bytes(document_bytes, byteorder="big"),
                    birthdate=birthdate_bytes.decode("utf-8"),
                    number=int.from_bytes(number_bytes, byteorder="big")
                )
                bets.append(bet)

            # mandamos al cliente que el batch fue procesado correctamente
            safe_socket.send_all(client_socket, b"\x00")

            #logger.info(action, logger.LogResult.success, "agency_id", agency_id, "batch_size", len(bets))
            return bets

        except Exception as e:
            logger.error(action, logger.LogResult.fail, "error", str(e))
            raise e
    
    def send_winners(self, client_socket: socket.socket, winners: list[Bet]):
        action = "send-winners"
        try:
            logger.info(action, logger.LogResult.in_progress, "count", len(winners))

            # buffer unico para todo el payload
            payload = bytearray()

            # cantidad de ganadores (4 bytes)
            payload.extend(len(winners).to_bytes(4, byteorder="big"))

            for winner in winners:
                name_bytes = winner.first_name.encode("utf-8")
                payload.extend(len(name_bytes).to_bytes(_NAME_LENGTH_SIZE, byteorder="big"))
                payload.extend(name_bytes)

                last_name_bytes = winner.last_name.encode("utf-8")
                payload.extend(len(last_name_bytes).to_bytes(_LAST_NAME_LENGTH_SIZE, byteorder="big"))
                payload.extend(last_name_bytes)

                payload.extend(winner.document.to_bytes(_DOCUMENT_SIZE, byteorder="big"))
                payload.extend(winner.birthdate.encode("utf-8"))
                payload.extend(winner.number.to_bytes(_NUMBER_SIZE, byteorder="big"))

            safe_socket.send_all(client_socket, bytes(payload))

            logger.info(action, logger.LogResult.success, "count", len(winners))

        except Exception as e:
            logger.error(action, logger.LogResult.fail, "error", str(e))
            raise e