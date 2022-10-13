# UniteHUD
Pokémon UNITE scoreboard HUD and extra tools running over captured game feeds.

#### For beta support, message me on [twitter](https://twitter.com/pidgy_)
----
### v1.0 Beta Released
- Available for download 👉 **https://unitehud.dev**

----

### Client (OBS)
![alt text](https://github.com/pidgy/unite/blob/master/data/client2.gif "Client")

### Server
![alt text](https://i.imgur.com/X9T7vpH.png "server")

### Architecture

- The server opens port 17069 by default as a Websocket and HTTP endpoint. 
- The client sends a GET request every second to the server and updates it's page.

#### Client Request
##### HTTP
```
GET 127.0.0.1:17069/http
```
##### WebSocket
```
GET 127.0.0.1:17069/ws
```

#### Server Response
##### HTTP/WebSocket
```
{
    "balls":0,
    "orange":{
        "team":"orange",
        "value":0
    },
    "purple":{
        "team":"purple",
        "value":0
    },
    "regis":["none","none","none"],
    "seconds":0,
    "self":{
    "team":"self",
        "value":0
    },
    "stacks":0,
    "started":false,
    "version":"v1.0beta"
}
```

### Note
- This project is currently in a beta state. 
- It would be possible for matching techniques to produce duplicated, unaccounted-for, and false postitive matches.
- Winner/Loser confidence is successful ~99% of the time.
- Score tracking is ~90% accurate, certain game mechanics (like rotom scoring points) are extremely difficult to process.
- Users are encouraged to report issues, or contribute where they can to help polish a final product.

# Testing
- - Head into Pokémon UNITE's Practice Mode and verify UniteHUD is capturing time/orbs/enemy score/self score.
- - Use the "Configure" button to verify the selection areas.

