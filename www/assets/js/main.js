const urlWS = "ws://127.0.0.1:17069/ws";
const urlHTTP = "http://127.0.0.1:17069/http";

const cached = {
    banners: {
        title: `UniteHUD`,
    },
    assets: {
        get img() { return 'assets/img/sprites'; },
    },
    img: {
        objectives: {
            get top() { return `${cached.assets.img}/regieleki.png`; },
            get central() { return `${cached.assets.img}/rayquaza.png`; },
            get bottom() {
                return [
                    `${cached.assets.img}/regice.png`,
                    `${cached.assets.img}/regirock.png`,
                    `${cached.assets.img}/registeel.png`,
                ];
            },
        },
    },
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
    "ready": true,
    "seconds": 10 * 60,
    "purple": { "value": 0, "kos": 0, "surrendered": false },
    "orange": { "value": 0, "kos": 0, "surrendered": false },
    "self": { "value": 0, "kos": 0 },
    "stacks": 0,
    "regis": ["none", "none", "none"], // purple, orange, none.
    "bottom": [], // {"name": "registeel", "team": "purple"}
    "events": [],
    "debug": false,
};


function animate() {
    if (intervals.should.shake) {
        $('.unitehud-banner-label').css('animation', 'shake 1s cubic-bezier(.36,.07,.19,.97) both');
    } else {
        $('.unitehud-banner-label').css('animation', 'none');
    }

    if (intervals.should.spin) {
        $('.banner-logo').css('animation', 'rotate-center 1s cubic-bezier(.36,.07,.19,.97) both');
    } else {
        $('.banner-logo').css('animation', 'none');
    }
}

function clear(err = '') {
    $('.unitehud-banner-label').html(err);

    $('.hud-banner').css('opacity', '.5');
    if (err.includes(cached.banners.title)) {
        $('.unitehud-banner-label').css('opacity', .25);
    } else {
        $('.debug-banner-labels').css('opacity', 0);
        $('.debug-banner.banner').css('opacity', 0);
    }

    $('.team-score-container').css('opacity', 0);
    $('.objectives-container').css('opacity', 0);
    $(`.objectives-circle.orange`).css('opacity', 0);
    $(`.objectives-circle.purple`).css('opacity', 0);
    $(`.objectives-circle.none`).css('opacity', 1);

    for (var i = 1; i <= 3; i++) {
        $(`.objectives-${i}.bottom`).filter("img").attr('src', cached.img.objectives.bottom[i]);
        $(`.objectives-${i}`).filter("img").css('opacity', .75);
        if (i == 1) {
            $(`.objectives-${i}.central`).filter("img").css('opacity', .75);
        }
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
        error: error,
    });
};

// Successfully connected to the UniteHUD application.
async function render(data) {
    if (data.debug) {
        debug.events.add(data.events);
    } else {
        $('.debug-banner-labels').css('opacity', 0);
        $('.debug-banner.banner').css('opacity', 0);
    }

    // User has not pressed "start".
    if (!data.ready) {
        return clear(`Press Start`);
    }

    // Render HUD.
    {
        // Match started
        if (data.match && !prev.match) {
            $('.unitehud-banner-label').html('');

            $('.hud-container').css('opacity', '0').animate(
                properties = {
                    opacity: '1',
                },
                duration = 10 * 1000,
                complete = () => {}
            );

            prev.match = data.match;
        }

        // Match ended fade away.
        if (!data.match && prev.match) {
            $('.hud-container').animate(
                properties = {
                    opacity: '0',
                },
                duration = 10 * 1000,
                complete = () => {}
            );

            prev.match = data.match;
        }

        // User has presseed "start", awaiting match detection.
        if (data.seconds == 0 && prev.seconds != 0) {
            return clear(`${cached.banners.title} <span>${data.version}</span>`);
        }
        prev.seconds = data.seconds;
    }

    // Render scores.
    {

        // Render purple score.
        {
            var phtml = `<div class="animated">${data.purple.value}</div>`;
            var p = data.regis.filter(x => x === "purple").length;
            if (p > 0) {
                phtml = `<div class="animated">${data.purple.value}</div> <span><i>max ${data.purple.value + p * 20}</i></span>`;
            }
            if (data.purple.surrendered) {
                phtml = `<div class="animated">SND</div>`;
            }
            $('.team-score.purple').html(phtml);

            // Check if purple team scored. 
            if (prev.purple && prev.purple.value != data.purple.value) {
                $('.team-score.purple .animated').css('animation', 'scored 1s cubic-bezier(.36,.07,.19,.97) both');
                prev.purple.value = data.purple.value;
            } else {
                $('.team-score.purple .animated').css('animation', 'none');
            }
        }

        // Render orange score.
        {
            var ohtml = `<div class="animated">${data.orange.value}</div>`;
            var o = data.regis.filter(x => x === "orange").length;
            if (o > 0) {
                ohtml = `<div class="animated">${data.orange.value}</div><span><i>max ${data.orange.value + o * 20}</i></span>`;
            }
            if (data.orange.surrendered) {
                ohtml = `<div class="animated">SND</div>`;
            }
            $('.team-score.orange').html(`${ohtml}`);

            // Check if orange team scored.
            if (prev.orange && prev.orange.value != data.orange.value) {
                $('.team-score.orange .animated').css('animation', 'scored 1s cubic-bezier(.36,.07,.19,.97) both');
                prev.orange.value = data.orange.value;
            } else {
                $('.team-score.orange .animated').css('animation', 'none');
            }
        }
    }

    // Render top objectives.
    {
        $('div')
            .filter('.hud-container.objectives-container.top')
            .children('img')
            .attr('src', cached.img.objectives.top);

        var elekis = {
            "none": ["none", "orange", "purple"],
            "purple": ["purple", "orange", "none"],
            "orange": ["orange", "purple", "none"],
        }

        for (var i in data.regis) {
            i = parseInt(i);

            if (data.regis[i] == "none") {
                $(`.objectives-${i+1}.top`).filter("img").css({
                    'opacity': .75,
                    'animation': 'none'
                });
            } else {
                $(`.objectives-${i+1}.top`).filter("img").css({
                    'opacity': 1,
                    'animation': 'secured 1s cubic-bezier(.36,.07,.19,.97) both'
                });

                if (data.regis[i] != prev.regis[i]) {
                    $(`.${data.regis[i]}-score span`).css('animation', 'scored 1s cubic-bezier(.36,.07,.19,.97) both');
                    prev.regis[i] = data.regis[i];
                }
            }

            $(`.objectives-${i+1}.top .objectives-circle.${elekis[data.regis[i]][0]}`).css('opacity', 1);
            $(`.objectives-${i+1}.top .objectives-circle.${elekis[data.regis[i]][1]}`).css('opacity', 0);
            $(`.objectives-${i+1}.top .objectives-circle.${elekis[data.regis[i]][2]}`).css('opacity', 0);
        }
    }

    // Render central objectives.
    {
        $('div')
            .filter('.hud-container.objectives-container.central')
            .children('img')
            .attr('src', cached.img.objectives.central);

        if (data.rayquaza) {
            $(`.objectives-1.central .objectives-circle.none`).css('opacity', 0);
            $(`.objectives-1.central .objectives-circle.${data.rayquaza}`).css('opacity', 1);
            $(`.objectives-1.central`).filter("img").css({
                'opacity': 1,
                'animation': 'secured 1s cubic-bezier(.36,.07,.19,.97) both'
            });
        } else {
            $(`.objectives-1.central .objectives-circle.purple`).css('opacity', 0);
            $(`.objectives-1.central .objectives-circle.orange`).css('opacity', 0);
            $(`.objectives-1.central .objectives-circle.none`).css('opacity', 1);
            $(`.objectives-1.central`).filter("img").css({
                'opacity': .75,
                'animation': 'none'
            });
        }
    }

    // Render bottom objectives.
    {
        for (var i = 0; i < 3; i++) {
            $(`.objectives-${i+1}.bottom .objectives-circle.purple`).css('opacity', 0);
            $(`.objectives-${i+1}.bottom .objectives-circle.orange`).css('opacity', 0);
            $(`.objectives-${i+1}.bottom .objectives-circle.none`).css('opacity', 0);

            if (data.bottom.length <= i) {
                $(`.objectives-${i+1}.bottom .objectives-circle.purple`).css('opacity', 0);
                $(`.objectives-${i+1}.bottom .objectives-circle.orange`).css('opacity', 0);
                $(`.objectives-${i+1}.bottom .objectives-circle.none`).css('opacity', 1);

                $(`.objectives-${i+1}.bottom`).filter("img").attr('src', cached.img.objectives.bottom[i]).css({
                    'opacity': .75,
                    'animation': 'none'
                });
            } else {
                var obj = data.bottom[i];
                $(`.objectives-${i+1}.bottom .objectives-circle.${obj.team}`).css('opacity', 1);

                $(`.objectives-${i+1}.bottom`).filter("img").attr('src', `${cached.assets.img}/${obj.name}.png`).css({
                    'opacity': 1,
                    'animation': 'secured 1s cubic-bezier(.36,.07,.19,.97) both'
                });
            }
        }
    }
}


// WebSocket connection handler.
function websocket() {
    var ws = new WebSocket(urlWS);

    ws.onmessage = (event) => {
        render(JSON.parse(event.data));
        ws.close();
    };

    ws.onerror = error;
}

$(document).ready(() => {
    var opacity = $('.hud-banner').css('opacity');
    $('.hud-banner').css('opacity', '0').animate(
        properties = { opacity: opacity, },
        duration = 5 * 1000,
        complete = () => { clear(); }
    );
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
            "ready": true,
            "seconds": 10 * 60,
            "purple": { "value": 0, "kos": 0, "surrendered": false },
            "orange": { "value": 0, "kos": 0, "surrendered": false },
            "self": { "value": 0, "kos": 0 },
            "stacks": 0,
            "regis": ["none", "none", "none"], // purple, orange, none.
            "bottom": [], // {"name": "registeel", "team": "purple"}
            "events": [],
            "debug": true,
            "match": true,
        };
    },
    ready: {
        get toggle() { return debug.data.ready = !debug.data.ready; },
    },
    time: {
        get finalstretch() { return debug.data.seconds = 120; },
    },
    score: {
        get purple() { return debug.data.purple.value += Math.floor(Math.random() * 100); },
        get orange() { return debug.data.orange.value += Math.floor(Math.random() * 100); },
    },
    surrender: {
        get purple() { return debug.data.purple.surrender = true; },
        get orange() { return debug.data.orange.surrender = true; },
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
    events: {
        get generate() {
            return debug.events.add([
                "[19:53:18] [Detect] [09:56] [Self] Holding 2 points",
                "[19:53:25] [Detect] [09:48] [Self] Holding 4 points",
                "[19:53:31] [Detect] [09:45] [Self] Holding 6 points",
                "[19:53:39] [Detect] [09:34] [Orange] [First] +4",
                "[19:53:42] [Detect] [09:34] [Purple] +2",
                "[19:53:44] [Detect] [09:34] [Self] Holding 11 points",
                "[19:53:45] [Detect] [09:34] [Purple] [Potential Duplicate] [max-gt-91] +2",
                "[19:53:45] [Detect] [09:34] [Purple] +2",
                "[19:53:49] [Detect] [09:24] [Purple] +5",
                "[19:53:52] [Detect] [09:21] [Orange] +5",
                "[19:53:59] [Detect] [09:20] [Orange] [Potential Duplicate] [long-delay] +5",
                "[19:53:59] [Detect] [09:20] [Orange] +5",
                "[19:54:00] [Detect] [09:13] [Purple] [Potential Duplicate] [long-delay] +5",
                "[19:54:00] [Detect] [09:13] [Purple] +5",
                "[19:54:03] [Detect] [09:11] [Purple] [Duplicate] [max-gt-91] +5",
                "[19:54:03] [Detect] [09:11] [Purple] [Duplicate] +5",
                "[19:54:14] [Detect] [08:59] [Self] Holding 16 points",
                "[19:54:29] [Detect] [08:49] [Self] Holding 17 points",
                "[19:54:38] [Detect] [08:34] [Orange] +14",
                "[19:54:41] [Detect] [08:34] [Orange] +26",
                "[19:54:41] [Detect] [08:34] [Self] Score option present (17)",
                "[19:54:44] [Detect] [08:34] [Self] Holding 0 points",
                "[19:54:44] [Detect] [08:34] [Orange] +19",
                "[19:54:45] [Detect] [08:34] [Self] +17 Confirming points scored 0s ago",
                "[19:54:45] [Detect] [08:34] [Purple] [Self] +17",
                "[19:54:48] [Detect] [08:25] [Purple] +8",
                "[19:54:51] [Detect] [08:23] [Purple] +27",
                "[19:54:53] [Detect] [08:19] [Purple] +1",
                "[19:54:56] [Detect] [08:17] [Purple] [Potential Duplicate] [max-gt-91] +1",
                "[19:54:56] [Detect] [08:17] [Purple] +1",
                "[19:54:58] [Detect] [08:15] [Purple] [Duplicate] [max-gt-91] +1",
                "[19:54:58] [Detect] [08:15] [Purple] [Duplicate] +1",
                "[19:55:01] [Detect] [08:12] [Self] Holding 5 points",
                "[19:55:09] [Detect] [08:10] [Self] Holding 10 points",
                "[19:55:31] [Detect] [07:45] [Self] Holding 15 points",
                "[19:55:35] [Detect] [07:38] [Self] Holding 17 points",
                "[19:56:01] [Detect] [07:12] [Self] Holding 21 points",
                "[19:56:36] [Detect] [06:37] [Purple] +6",
                "[19:56:39] [Detect] [06:37] [Purple] [Missed] +0",
                "[19:56:41] [Detect] [06:37] [Self] Holding 26 points",
                "[19:56:41] [Detect] [06:37] [Purple] +24",
                "[19:56:44] [Detect] [06:29] [Self] Score option present (26)",
                "[20:03:26] [Detect] [Game] Match ended",
                "[20:03:26] [Detect] [Purple] 238 [+0 Regielekis] [+0 Regices] [+0 Regirocks] [+0 Registeels] [+0 Rayquazas]",
                "[20:03:26] [Detect] [Orange] 251 [+2 Regielekis] [+0 Regices] [+1 Regirock] [+1 Registeel] [+0 Rayquazas]",
                "[20:03:26] [Detect] [Self] 97",
                "[20:03:35] [UI] Main stage: StageRunning",
                "[20:03:36] [UI] Main stage: StageInactive",
                "[20:20:42] [Detect] Averaging 157.322µs / 0.9kB latency",
                "[20:25:46] [Stats] Matched image template statistics",
            ]);
        },
        add: (events) => {
            $('.debug-banner-labels').css('opacity', .9);
            $('.debug-banner.banner').css('opacity', .9);

            const unique = events
                .filter(event => !prev.events.includes(event))
                .filter(event => !event.includes("[UI]") && !event.includes(" Holding ") && !event.includes(" Confirming ") &&
                    (event.includes("[Purple]") || event.includes("[Orange]") || event.includes("[Game]") || event.includes("[Self]")));
            if (!unique || unique.length == 0) {
                return;
            }
            if (unique.toString() == prev.events.toString()) {
                return;
            }

            prev.events.push(...unique);
            prev.events = prev.events.slice(-1000);

            unique.forEach((event) => {
                var img = "";
                /*
                if (event.includes("[Purple] +")) {
                    img = `<img class="debug-label-banner-logo" src="assets/img/aeos-purple.png">`
                } else if (event.includes("[Orange] +")) {
                    img = `<img class="debug-label-banner-logo" src="assets/img/aeos-orange.png">`
                } else if (event.includes("[Purple] [Self] +")) {
                    img = `<img class="debug-label-banner-logo" src="assets/img/aeos-purple.png">`
                } else if (event.includes(" Regielekis")) {
                    img = `<img class="debug-label-banner-logo" src="assets/img/pokemonunite.png">`
                } else if (event.includes(" Regieleki ")) {
                    img = `<img class="debug-label-banner-logo" src="assets/img/regieleki.png">`
                } else if (event.includes(" Registeel ")) {
                    img = `<img class="debug-label-banner-logo" src="assets/img/registeel.png">`
                } else if (event.includes(" Regice ")) {
                    img = `<img class="debug-label-banner-logo" src="assets/img/regice.png">`
                } else if (event.includes(" Regirock ")) {
                    img = `<img class="debug-label-banner-logo" src="assets/img/regirock.png">`
                } else if (event.includes("Defeated")) {
                    img = `<img class="debug-label-banner-logo" src="assets/img/unscored.png">`
                } else if (event.includes("[Self]")) {
                    img = `<img class="debug-label-banner-logo" src="assets/img/unscored.png">`
                } else if (event.includes(" Rayquaza ")) {
                    img = `<img class="debug-label-banner-logo" src="assets/img/rayquaza.png">`
                } else if (event.includes("[Game]")) {
                    img = `<img class="debug-label-banner-logo" src="assets/img/pokemonunite.png">`
                } else if (event.includes("[Purple]")) {
                    img = `<img class="debug-label-banner-logo" src="assets/img/ko_purple.png">`
                } else if (event.includes("[Orange]")) {
                    img = `<img class="debug-label-banner-logo" src="assets/img/ko_orange.png">`
                }
                img = ``;
                */

                $(`.debug-banner-labels`).append(`<li style="opacity:0">${img} ${event}</li>`);

                // const size = $(`.debug-banner-labels`).children().length;
                // if (size > max) {
                //     var rem = size - max;
                //     $(`.debug-banner-labels`).children('li').each(function() {
                //         if (rem > 0) {
                //             this.remove();
                //         }
                //         rem--;
                //     });
                // }

                $(`.debug-banner-labels`).children('li').each(function() {
                    const child = $(this);
                    child.animate({
                        opacity: '.9'
                    }, 750, () => {});
                }).promise().done($('.float-top-right').scrollTop($('.float-top-right')[0].scrollHeight));
            });
        }
    },
};