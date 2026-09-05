import threading

class ReadWriteLock:

    def __init__(self):
        self._condition = threading.Condition(threading.Lock())
        self._readers = 0

    def acquire_read(self):
        with self._condition:
            self._readers += 1

    def release_read(self):
        with self._condition:
            self._readers -= 1
            if self._readers == 0:
                self._condition.notify_all()

    def acquire_write(self):
        self._condition.acquire()
        while self._readers > 0:
            self._condition.wait()

    def release_write(self):
        self._condition.release()
