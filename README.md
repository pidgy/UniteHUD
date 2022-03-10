# unite
Pokemon Unite scoreboard HUD and extra tools running over captured game feeds using OpenCV with a Gio interface.

### Note
##### This project is currently in early Alpha stages. 
##### A handful of matching techniques produce duplicated, unaccounted-for, and false postitive matches.
##### Users are encouraged to report issues, or contribute where they can to help polish a final product.

----

### Client (OBS Live)
![alt text](https://github.com/pidgy/unite/blob/master/data/client.gif "Client")

### Server
![alt text](https://github.com/pidgy/unite/blob/master/data/server.gif "server")

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

### TODO
- Detect "First Goal" messages.
- Optimize CPU usage (smaller areas, better image matching)
- Wiki for tutorial's
