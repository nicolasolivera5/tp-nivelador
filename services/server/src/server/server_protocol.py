import logger
import socket
import safe_socket
from services.server.src_frozen.lottery import bet

_BIRTHDATE_SIZE = 10
_NUMBER_SIZE = 4
_DOCUMENT_SIZE = 4
_AGENCY_ID_SIZE = 2
_NAME_LENGTH_SIZE = 1
_LAST_NAME_LENGTH_SIZE = 1

class ServerProtocol:
    def receive_bet(self, client_socket: socket.socket):

        # ID de la Agencia (2 bytes)
        agency_id_bytes = safe_socket.recv_all(client_socket, _AGENCY_ID_SIZE)
        agency_id = int.from_bytes(agency_id_bytes, byteorder="big")

        # tamanio del nombre (1 byte)
        name_length_bytes = safe_socket.recv_all(client_socket, _NAME_LENGTH_SIZE)
        name_length = int.from_bytes(name_length_bytes, byteorder="big")

        # si la longitud del nombre es 0, terminaron las apuestas de esta agencia
        if name_length == 0:
            return None

        # nombre
        name_bytes = safe_socket.recv_all(client_socket, name_length)

        # apellido
        last_name_length_bytes = safe_socket.recv_all(client_socket, _LAST_NAME_LENGTH_SIZE)
        last_name_length = int.from_bytes(last_name_length_bytes, byteorder="big")
        last_name_bytes = safe_socket.recv_all(client_socket, last_name_length)

        # documento (4 bytes binarios)
        document_bytes = safe_socket.recv_all(client_socket, _DOCUMENT_SIZE)

        # fecha de nacimiento (10 bytes texto)
        birthdate_bytes = safe_socket.recv_all(client_socket, _BIRTHDATE_SIZE)

        # apuesta (4 bytes binarios)
        number_bytes = safe_socket.recv_all(client_socket, _NUMBER_SIZE)

        return bet.Bet(
            agency_id=agency_id,
            first_name=name_bytes.decode("utf-8"),
            last_name=last_name_bytes.decode("utf-8"),
            document=int.from_bytes(document_bytes, byteorder="big"),
            birthdate=birthdate_bytes.decode("utf-8"),
            number=int.from_bytes(number_bytes, byteorder="big")
        )
    
    def send_winners(self, client_socket: socket.socket, winners: list[bet.Bet]):

        # envio la cantidad de ganadores (4 bytes)
        safe_socket.send_all(client_socket, len(winners).to_bytes(4, byteorder="big"))

        for winner in winners:
            name_bytes = winner.first_name.encode("utf-8")
            safe_socket.send_all(client_socket, len(name_bytes).to_bytes(_NAME_LENGTH_SIZE, byteorder="big"))
            safe_socket.send_all(client_socket, name_bytes)

            last_name_bytes = winner.last_name.encode("utf-8")
            safe_socket.send_all(client_socket, len(last_name_bytes).to_bytes(_LAST_NAME_LENGTH_SIZE, byteorder="big"))
            safe_socket.send_all(client_socket, last_name_bytes)

            safe_socket.send_all(client_socket, winner.document.to_bytes(_DOCUMENT_SIZE, byteorder="big"))
            safe_socket.send_all(client_socket, winner.birthdate.encode("utf-8"))
            safe_socket.send_all(client_socket, winner.number.to_bytes(_NUMBER_SIZE, byteorder="big"))