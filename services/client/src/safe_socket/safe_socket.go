package safe_socket

import "io"

//TODO: Complete with a short-read/short-write tolerant implementation

func SendAll(socket io.Writer, bytes []byte) error {
	
	for n := 0; n < len(bytes); {
		m, err := socket.Write(bytes[n:])
		if err != nil {
			return err
		}
		n += m
	}
	return nil
}

func RecvAll(socket io.Reader, size int) ([]byte, error) {
	buff := make([]byte, size)
	n := 0

	for n < size {
		m, err := socket.Read(buff[n:])
		if err != nil {
			return nil, err
		}
		n += m
	}

	return buff, nil
	
}
