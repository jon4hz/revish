# revish

Revish is a reverse ssh server/client.

## About
The client starts a local ssh server on the target machine, starts a client connection with the remote server and forwards the local server port to the remote server.  
The remote server also has an instance of wishlist running, to get an easy accessible list of all connected reverse sessions.

## Usage
```
# TODO
```

## ToDo
[ ] add more cli options for customization  
[ ] implement port negotiation if the remote server can use the forwarded port  
[ ] support sftp   

## Acknowledgments
- https://github.com/Fahrj/reverse-ssh
- https://github.com/charmbracelet/wish
- https://github.com/charmbracelet/wishlist