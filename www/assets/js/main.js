const urlWS = "ws://127.0.0.1:17069/ws";
const urlHTTP = "http://127.0.0.1:17069/http";

const cached = {
    messages: {
        ready: `UniteHUD`,
    },
    images: {
        regis: ['assets/img/regice.png', 'assets/img/regirock.png', 'assets/img/registeel.png'],
    }
}

const sync = function(func, delay) {
    var wait, id;

    wait = () => {
        func();
        id = setTimeout(wait, delay);
    };

    id = setTimeout(wait, delay);

    return () => { clearTimeout(id); };
};

var intervals = {
    loading: {
        _text: ["", ".", "..", "..."],
        _idx: 0,
        get next() { this._idx = ++this._idx % this._text.length; return this._text[this._idx]; },
    },
    _shake: 0,
    _spin: 0,
    should: {
        get shake() { if (++intervals._shake == 4) { intervals._shake = 0; return true } return false; },
        get spin() { if (++intervals._spin == 4) { intervals._spin = 0; return true } return false; },
    },
};

var prev = {
    "profile": "player",
    "version": "debug",
    "started": true,
    "seconds": 10 * 60,
    "purple": { "value": 0, "kos": 0 },
    "orange": { "value": 0, "kos": 0 },
    "self": { "value": 0, "kos": 0 },
    "stacks": 0,
    "regis": ["none", "none", "none"], // purple, orange, none.
    "bottom": [], // {"name": "registeel", "team": "purple"}
};


function animate() {
    if (intervals.should.shake) {
        $('.error').css('animation', 'shake 1s cubic-bezier(.36,.07,.19,.97) both');
    } else {
        $('.error').css('animation', 'none');
    }

    if (intervals.should.spin) {
        $('.logo').css('animation', 'rotate-center 1s cubic-bezier(.36,.07,.19,.97) both');
    } else {
        $('.logo').css('animation', 'none');
    }
}

function clear(err = '') {
    $('.error').css('opacity', '.9');
    $('.error').html(err);
    if (err.includes(cached.messages.ready)) {
        $('.error').css('opacity', .25);
    }

    $('.purple').css('opacity', 0);
    $('.orange').css('opacity', 0);
    $('.self').css('opacity', 0);
    $('.objectives-top').css('opacity', 0);
    $('.objectives-bottom').css('opacity', 0);
    $('.objectives-central').css('opacity', 0);

    $(`.objectives-central-1 .objectives-central-circle-purple`).css('opacity', 0);
    $(`.objectives-central-1 .objectives-central-circle-orange`).css('opacity', 0);
    $(`.objectives-central-1 .objectives-central-circle-none`).css('opacity', 1);
    $(`.objectives-central-img-1`).css('opacity', .75);

    for (var i = 1; i <= 3; i++) {
        $(`.objectives-bottom-${i} .objectives-bottom-circle-purple`).css('opacity', 0);
        $(`.objectives-bottom-${i} .objectives-bottom-circle-orange`).css('opacity', 0);
        $(`.objectives-bottom-img-${i}`).attr('src', cached.images.regis[i]);
        $(`.objectives-bottom-img-${i}`).css('opacity', .75);

        $(`.objectives-img-${i}`).css('opacity', .75);
    }
}

function error(err) {
    switch (typeof err) {
        case "string":
            clear(`${err}`);
            break;
        default:
            clear(`Connecting${intervals.loading.next}`);
            break;
    }

    return animate();
}

// HTTP connection handler.
function http() {
    $.ajax({
        type: 'GET',
        dataType: 'json',
        url: urlHTTP,
        timeout: 1000,
        success: function(data, status) {
            render(data);
        },
        error: error(),
    });
};

// Successfully connected to the UniteHUD application.
async function render(data) {
    // User has not pressed "start".
    if (!data.started) {
        return error(`Press Start`);
    }

    // User has presseed "start", awaiting match detection.
    if (data.seconds == 0) {
        return clear(`${cached.messages.ready} <span>${data.version}</span>`);
    }

    // Render HUD.
    {
        $('.error').html('');
        $('.purple').css('opacity', 1);
        $('.orange').css('opacity', 1);
        $('.objectives-top').css('opacity', 1);
        $('.objectives-bottom').css('opacity', 1);
        $('.objectives-central').css('opacity', 1);
    }

    // Render scores.
    {
        var pspan = "";
        var p = data.regis.filter(x => x === "purple").length;
        if (p > 0) {
            pspan = ` <span><i>max ${data.purple.value + p* 20}</i></span>`;
        }
        $('.purplescore').html(`<div class="animated">${data.purple.value}</div>${pspan}`);

        var ospan = "";
        var o = data.regis.filter(x => x === "orange").length;
        if (o > 0) {
            ospan = ` <span><i>max ${data.orange.value + o * 20}</i></span>`;
        }

        $('.orangescore').html(`<div class="animated">${data.orange.value}</div>${ospan}`);

        // Check if orange team scored.
        if (prev.orange && prev.orange.value != data.orange.value) {
            $('.orangescore .animated').css('animation', 'scored 1s cubic-bezier(.36,.07,.19,.97) both');
            prev.orange.value = data.orange.value;
        } else {
            $('.orangescore .animated').css('animation', 'none');
        }

        // Check if purple team scored. 
        if (prev.purple && prev.purple.value != data.purple.value) {
            $('.purplescore .animated').css('animation', 'scored 1s cubic-bezier(.36,.07,.19,.97) both');
            prev.purple.value = data.purple.value;
        } else {
            $('.purplescore .animated').css('animation', 'none');
        }
    }

    // Render top objectives.
    {
        var elekis = {
            "none": ["none", "orange", "purple"],
            "purple": ["purple", "orange", "none"],
            "orange": ["orange", "purple", "none"],
        }

        for (var i in data.regis) {
            i = parseInt(i);

            if (data.regis[i] == "none") {
                $(`.objectives-img-${i+1}`).css('opacity', .75);
                $(`.objectives-img-${i+1}`).css('animation', 'none');
            } else {
                $(`.objectives-img-${i+1}`).css('opacity', 1);
                $(`.objectives-img-${i+1}`).css('animation', 'secured 1s cubic-bezier(.36,.07,.19,.97) both');

                if (data.regis[i] != prev.regis[i]) {
                    $(`.${data.regis[i]}score span`).css('animation', 'scored 1s cubic-bezier(.36,.07,.19,.97) both');
                    prev.regis[i] = data.regis[i];
                }
            }

            $(`.objectives-${i+1} .objectives-circle-${elekis[data.regis[i]][0]}`).css('opacity', 1);
            $(`.objectives-${i+1} .objectives-circle-${elekis[data.regis[i]][1]}`).css('opacity', 0);
            $(`.objectives-${i+1} .objectives-circle-${elekis[data.regis[i]][2]}`).css('opacity', 0);
        }
    }

    // Render central objectives.
    {
        $(`.objectives-central-1 .objectives-central-circle-purple`).css('opacity', 0);
        $(`.objectives-central-1 .objectives-central-circle-orange`).css('opacity', 0);
        $(`.objectives-central-1 .objectives-central-circle-none`).css('opacity', 1);

        if (data.rayquaza) {
            $(`.objectives-central-1 .objectives-central-circle-none`).css('opacity', 0);
            $(`.objectives-central-1 .objectives-central-circle-${data.rayquaza}`).css('opacity', 1);
            $(`.objectives-central-img-1`).css('opacity', 1);
            $(`.objectives-central-img-1`).css('animation', 'secured 1s cubic-bezier(.36,.07,.19,.97) both');
        } else {
            $(`.objectives-central-img-1`).css('opacity', .75);
            $(`.objectives-central-img-1`).css('animation', 'none');
        }
    }

    // Render bottom objectives.
    {
        for (var i = 0; i < 3; i++) {
            $(`.objectives-bottom-${i+1} .objectives-bottom-circle-purple`).css('opacity', 0);
            $(`.objectives-bottom-${i+1} .objectives-bottom-circle-orange`).css('opacity', 0);
            $(`.objectives-bottom-${i+1} .objectives-bottom-circle-none`).css('opacity', 0);

            if (data.bottom.length <= i) {
                $(`.objectives-bottom-${i+1} .objectives-bottom-circle-purple`).css('opacity', 0);
                $(`.objectives-bottom-${i+1} .objectives-bottom-circle-orange`).css('opacity', 0);
                $(`.objectives-bottom-${i+1} .objectives-bottom-circle-none`).css('opacity', 1);
                $(`.objectives-bottom-img-${i+1}`).attr('src', cached.images.regis[i]);
                $(`.objectives-bottom-img-${i+1}`).css('opacity', .75);
                $(`.objectives-bottom-img-${i+1}`).css('animation', 'none');
            } else {
                var obj = data.bottom[i];
                $(`.objectives-bottom-${i+1} .objectives-bottom-circle-${obj.team}`).css('opacity', 1);
                $(`.objectives-bottom-img-${i+1}`).attr('src', `assets/img/${obj.name}.png`);
                $(`.objectives-bottom-img-${i+1}`).css('opacity', '1');
                $(`.objectives-bottom-img-${i+1}`).css('animation', 'secured 1s cubic-bezier(.36,.07,.19,.97) both');
            }
        }
    }
}

// WebSocket connection handler.
function websocket() {
    var ws = new WebSocket(urlWS);
    ws.onmessage = function(event) {
        render(JSON.parse(event.data));
        ws.close();
    };
    ws.onerror = error;

    // setTimeout(() => {
    //     if (ws.readyState == WebSocket.CONNECTING) {
    //         ws.close();
    //     }
    // }, 500);
}

$(document).ready(() => {
    clear();

    switch (true) {
        case window.location.search.includes('debug'):
            sync(debug.start, 1000);
            break;
        case window.location.search.includes('http'):
            console.log(`[UniteHUD] creating http connection to ${urlHTTP}`);
            sync(http, 1000);
            break;
        default:
            console.info(`[UniteHUD] creating websocket connection to ${urlWS} (add "?http" to connect to the http endpoint)`);
            sync(websocket, 1000);
            break;
    }
});

const debug = {
    get error() { return error(); },
    get prev() { return console.info(JSON.stringify(prev, null, 2)); },
    get start() { return () => { debug.data ? render(debug.data) : debug.reset; }; },
    get object() { return console.info(debug.data); },
    get json() { return console.info(JSON.stringify(debug.data, null, 2)); },
    get reset() {
        $('body').css('background-image', 'url("assets/img/sample-bg.png")');

        return debug.data = {
            "profile": "player",
            "version": debug,
            "started": true,
            "seconds": 10 * 60,
            "purple": { "value": 0, "kos": 0 },
            "orange": { "value": 0, "kos": 0 },
            "self": { "value": 0, "kos": 0 },
            "stacks": 0,
            "regis": ["none", "none", "none"], // purple, orange, none.
            "bottom": [], // {"name": "registeel", "team": "purple"}
        };
    },
    started: {
        get toggle() { return debug.data.started = !debug.data.started; },
    },
    time: {
        get finalstretch() { return debug.data.seconds = 120; },
    },
    score: {
        get purple() { return debug.data.purple.value += Math.floor(Math.random() * 100); },
        get orange() { return debug.data.orange.value += Math.floor(Math.random() * 100); },
    },
    objectives: {
        top: {
            get purple() { return debug.data.regis[debug.data.regis.filter(x => x !== "none").length] = "purple"; },
            get orange() { return debug.data.regis[debug.data.regis.filter(x => x !== "none").length] = "orange"; },
            get clear() { return debug.data.regis = ["none", "none", "none"]; },
        },
        bottom: {
            get purple() { return debug.data.bottom.push({ "name": ["regirock", "registeel", "regice"][Math.floor(Math.random() * 3)], "team": "purple" }); },
            get orange() { return debug.data.bottom.push({ "name": ["regirock", "registeel", "regice"][Math.floor(Math.random() * 3)], "team": "orange" }); },
            get clear() { return debug.data.bottom = []; },
        },
        central: {
            get purple() { return debug.data.rayquaza = "purple"; },
            get orange() { return debug.data.rayquaza = "orange"; },
            get clear() { return debug.data.rayquaza = ""; },
        },
    },
};