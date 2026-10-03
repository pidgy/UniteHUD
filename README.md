# <img src="https://github.com/pidgy/UniteHUD/blob/master/assets/icon/icon.png" width="42" align="left" style="margin-right: 12px;"> UniteHUD

### Pokémon UNITE Scoreboard & HUD Overlay

**UniteHUD** is a real-time scoreboard and HUD overlay for [Pokémon UNITE](https://www.pokemonunite.com/), providing live match information, objective tracking, scoring data, and customizable overlays.

<p align="center">
  <a href="https://unitehud.dev">
    <img src="https://img.shields.io/badge/Download-UniteHUD.dev-5865F2?style=for-the-badge" alt="Download UniteHUD">
  </a>
  <a href="https://pkg.go.dev/github.com/pidgy/unitehud">
    <img src="https://img.shields.io/badge/Go-Reference-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go Reference">
  </a>
</p>

---

## ✨ Features

* 📊 **Live scoreboard tracking**
* 🎯 **Objective tracking**
* ⚽ **Score and KO tracking**
* 🖥️ **Customizable HUD overlays**
* 🎨 **Configurable capture regions**
* 🔌 **HTTP & WebSocket API**
* ⚡ **Real-time match events**
* 🧪 **Practice Mode testing support**

---

## 📥 Download

### **[→ Download UniteHUD](https://unitehud.dev)**

Get the latest version and setup instructions from the official website.

---

## ☕ Support

If UniteHUD has been useful to you, consider supporting development!

<p align="center">
  <a href="https://www.buymeacoffee.com/pidgy">
    <img src="https://i.imgur.com/TNfrDMT.png" width="100" alt="Buy Me a Coffee">
  </a>
</p>

---

# 🖥️ Screenshots & Demos

## Client UI

The UniteHUD client provides the interface for configuring and monitoring the overlay.

<p align="center">
  <img src="https://github.com/pidgy/unitehud/blob/master/.github/data/v2-ui.gif" alt="UniteHUD Client UI">
</p>

## Overlay HUD

Live scoreboard information displayed directly over gameplay.

<p align="center">
  <img src="https://github.com/pidgy/unitehud/blob/master/.github/data/v2-hud.gif" alt="UniteHUD Overlay HUD">
</p>

## Customizable Configuration

Configure capture regions and customize how UniteHUD interacts with the game.

<p align="center">
  <img src="https://github.com/pidgy/unitehud/blob/master/.github/data/v2-projector.gif" alt="UniteHUD Configuration">
</p>

## Objective Tracking

UniteHUD tracks objectives and their associated teams throughout the match.

<p align="center">
  <img src="https://github.com/pidgy/unitehud/blob/master/.github/data/v2-registeel.gif" alt="Registeel Tracking">
  <img src="https://github.com/pidgy/unitehud/blob/master/.github/data/v2-regieleki.gif" alt="Regieleki Tracking">
</p>

---

# 🏗️ Architecture

UniteHUD uses a lightweight local HTTP/WebSocket server to communicate between the capture process and the client.

```text
┌─────────────────────┐
│    Pokémon UNITE    │
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│      UniteHUD       │
│   Capture / Parser  │
└──────────┬──────────┘
           │
           ▼
┌──────────────────────────────┐
│ Local HTTP / WebSocket Server│
│         :17069               │
└──────────────┬───────────────┘
               │
        ┌──────┴──────┐
        ▼             ▼
     HTTP GET      WebSocket
        │             │
        └──────┬──────┘
               ▼
┌──────────────────────────────┐
│          Client UI            │
│       Overlay / HUD           │
└──────────────────────────────┘
```

The server listens on **port `17069`** by default and exposes both HTTP and WebSocket endpoints.

### Client → Server

#### HTTP

```http
GET 127.0.0.1:17069/http
```

#### WebSocket

```http
GET 127.0.0.1:17069/ws
```

---

# 📡 Server Response

Both HTTP and WebSocket endpoints provide match state in JSON format.

```json
{
    "purple": {
        "team": "purple",
        "value": 254,
        "kos": 12
    },
    "orange": {
        "team": "orange",
        "value": 367,
        "kos": 21
    },
    "self": {
        "team": "self",
        "value": 43
    },
    "seconds": 59,
    "balls": 34,
    "regis": [
        "orange",
        "purple",
        "orange"
    ],
    "bottom": [
        {
            "name": "regice",
            "team": "orange",
            "time": 1676760349
        },
        {
            "name": "regirock",
            "team": "purple",
            "time": 1676760390
        },
        {
            "name": "registeel",
            "team": "orange",
            "time": 1676760391
        }
    ],
    "started": true,
    "stacks": 3,
    "defeated": [
        421,
        342,
        120
    ],
    "match": true,
    "config": false,
    "profile": "player",
    "version": "v1.1",
    "final_objective": "orange",
    "events": [
        "[2:00] Defeated with points",
        "[1:45] Groudon orange secure"
    ]
}
```

---

# ⚠️ Accuracy & Limitations

UniteHUD relies on visual matching and game-state detection, so some edge cases can produce inaccurate results.

| System                      | Approx. Accuracy |
| --------------------------- | ---------------: |
| 🏆 Winner / Loser Detection |         **~99%** |
| 📊 Score Tracking           |         **~90%** |

### Known limitations

* Matching techniques can occasionally produce **duplicate, unaccounted-for, or false-positive matches**.
* Certain game mechanics can make score detection particularly difficult.
* Mechanics such as **Rotom scoring points** may be difficult to process accurately.
* Accuracy can vary depending on capture configuration and in-game conditions.

If you encounter an issue, please consider **reporting it or contributing a fix**. Every edge case helps make UniteHUD more reliable. 🛠️

---

# 🧪 Testing

The easiest way to test UniteHUD is through **Pokémon UNITE Practice Mode**.

### Recommended checklist

1. Enter **Practice Mode** in Pokémon UNITE.
2. Launch UniteHUD.
3. Verify that UniteHUD is correctly capturing:

   * ⏱️ Match time
   * ⚽ Aeos Energy / orbs
   * 🏆 Enemy score
   * 👤 Your score
4. Open **Configure**.
5. Verify that all capture/selection areas are positioned correctly.
6. Test objective detection and scoreboard updates.

---

# 🤝 Contributing

UniteHUD benefits from testing, bug reports, and contributions from the community.

If you find an issue or an edge case, please report it with as much information as possible. Contributions that improve detection accuracy, reliability, or usability are welcome.

---

<p align="center">
  <sub>Built for the Pokémon UNITE community 🎮</sub>
</p>
