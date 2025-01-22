import socket
from threading import Thread


from socket import socket
s = socket()
s.connect(("localhost", 8080))

# Compose the message/HTTP request we want to send to the server
msgPart1 = b"GET /index.html HTTP/1.1\r\nHost: website1\r\n\r\nGET /index.html HTTP/1.1\r\nHost: website1\r\n\r\n"

# Send out the request
s.sendall(msgPart1)

# Listen for response and print it out

print (s.recv(4096))

# s.close()


# def send_request(msg, host, port):
#     s = socket.socket()
#     s.connect((host, port))
#     s.sendall(msg)
#     response = s.recv(4096)
#     print()
#     print(response)
#     print()

# msgPart1 = b"GET / HTTP/1.1\r\nHost: website1\r\nUser-Agent: gotest\r\nConnection: close\r\n\r\n"
# msgPart2 = b"GET / HTTP/1.1\r\nHost: website2\r\nUser-Agent: gotest\r\nConnection: close\r\n\r\n"

# thread1 = Thread(target=send_request, args=(msgPart1, "localhost", 8080))
# thread2 = Thread(target=send_request, args=(msgPart2, "localhost", 8080))

# thread1.start()
# thread2.start()

# thread1.join()
# thread2.join()
