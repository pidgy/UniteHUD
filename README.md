# unite
Pokemon Unite scoreboard HUD and extra tools running over captured game feeds using the OpenCV video processing API.


### Client (OBS Live)
![alt text](https://github.com/pidgy/unite/blob/main/data/client.gif "Client")

### Server
![alt text](https://github.com/pidgy/unite/blob/main/data/server.gif "server")

### Architecture

- The server opens port 17069 by default as a Websocket and HTTP endpoint. 
- The client sends a GET request every second to the server and updates it's page.

#### Client Request
```
GET 127.0.0.1:17069/http
```

#### Server Response
```
{
    "orange": {
        "team": "orange",
        "value": 52
    },
    "purple": {
        "team": "purple",
        "value": 46
    },
    "seconds": 389,
    "self": {
        "team": "self",
        "value": 0
    }
}
```
