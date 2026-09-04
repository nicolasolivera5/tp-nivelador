import socket

# TODO: Complete with a short-read/short-write tolerant implementation


def recv_all(socket: socket.socket, size):

    bytes_recib = 0
    data = b""
    while bytes_recib < size:
        chunk = socket.recv(size - bytes_recib)
        if not chunk:
            raise Exception("Socket closed before receiving all data")
        data += chunk
        bytes_recib += len(chunk)
    return data


def send_all(socket: socket.socket, bytes):

    while len(bytes) > 0:
        bytes_sent = socket.send(bytes)
        bytes = bytes[bytes_sent:]

    return None