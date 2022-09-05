# UniteHUD
Pokémon UNITE scoreboard HUD and extra tools running over captured game feeds.

#### For beta support, message me on [Twitter](https://twitter.com/pidgy_)
----
### v0.9.1 Beta
- [Download](https://github.com/pidgy/unitehud/releases/download/v0.9.1-beta/UniteHUD_0.9_Installer.exe)
- Full monitor or Custom window capturing.
- WebSocket implementation to bypass CORS issues on OBS.

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

### Note
- This project is currently in a beta state. 
- It would be possible for matching techniques to produce duplicated, unaccounted-for, and false postitive matches.
- Winner/Loser confidence is successful ~99% of the time.
- Score tracking is ~90% accurate, certain game mechanics (like rotom scoring points) are extremely difficult to process.
- Users are encouraged to report issues, or contribute where they can to help polish a final product.

### Install/Setup (OBS) 
- Download the latest installer from the [Release](https://github.com/pidgy/unitehud/releases/) page.
- Start UniteHUD, select the "obs" button on the top right corner, and copy the directory URL from file explorer.
- Create a new browser source in OBS and check "Local File", browse to the directory URL and select "index.html".
- Right click your switch capture source in OBS and select "Fullscreen Projector (Source)"
- Alt+Tab back into UniteHUD and select the "Configure" button.
- Under Capture Window on the bottom-right hand side, select "Fullscreen Projector (Source)"
- From the Configure screen, select the Default Button to auto calibrate scoring areas.
- Select "Save"
- Select "Start"

- Optional 
- - Head into Pokémon UNITE's Practice Mode and verify UniteHUD is capturing time/orbs/enemy score/self score.
- - You may also use the "Configure" button to verify the selection areas.

