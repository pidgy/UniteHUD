const version = "v1.1";
const urlWS = "ws://127.0.0.1:17069/ws";
const urlHTTP = "http://127.0.0.1:17069/http";

var loaders = [".", "..", "..."];
var index = 0;
var loggedError = false;

var lastShake = 0;

function clear(err = '') {
    $('.purple').css('opacity', 0);
    $('.orange').css('opacity', 0);
    $('.self').css('opacity', 0);
    $('.regis').css('opacity', 0);
    $('.regis-bottom').css('opacity', 0);
    $('.rayquaza').css('opacity', 0);
    $('.error').css('opacity', '.9');
    $('.error').html(err);

    $(`.rayquaza-1 .rayquaza-circle-purple`).css('opacity', 0);
    $(`.rayquaza-1 .rayquaza-circle-orange`).css('opacity', 0);
    $(`.rayquaza-1 .rayquaza-circle-none`).css('opacity', 1);

    for (var i = 1; i <= 3; i++) {
        $(`.regis-bottom-${i} .regis-bottom-circle-purple`).css('opacity', 0);
        $(`.regis-bottom-${i} .regis-bottom-circle-orange`).css('opacity', 0);
        $(`.regis-bottom-img-${i}`).attr('src', 'assets/img/objective.png');
    }
}

function error(err) {
    switch (typeof err) {
        case "string":
            clear(`${err}`);

            if (!loggedError) {
                loggedError = true

                console.error(`${err}`);

                setTimeout(() => {
                    loggedError = false;
                }, 3600000);
            }
            break;
        default:
            clear(`Connecting${loaders[index]}`);
    }

    index = (index + 1) % loaders.length;

    return shake();
}

function http() {
    $.ajax({
        type: 'GET',
        dataType: 'json',
        url: urlHTTP,
        timeout: 1000,
        success: function(data, status) {
            success(data);
        },
        error: error(),
    });
};

function occurences(str, of) {
    var o = 0;
    for (var i in str) {
        if (str[i] == of) {
            o++;
        }
    }
    return o;
}

function shake() {
    lastShake++;
    if (lastShake == 5) {
        $('.error').css('animation', 'shake 1s cubic-bezier(.36,.07,.19,.97) both');
        $('.logo').css('animation', 'shake 1s cubic-bezier(.36,.07,.19,.97) both');
        lastShake = 0;
    } else {
        $('.error').css('animation', 'none');
        $('.logo').css('animation', 'none');
    }
}

function success(data) {
    loggedError = false;

    console.log(JSON.stringify(data))

    if (data.profile != "player") {
        error(`Invalid profile (${data.profile})`);
        return shake();
    }

    if (!data.started) {
        clear(`Press Start`);
        return shake();
    }

    if (data.seconds > 0) {
        $('.purple').css('opacity', 1);
        $('.orange').css('opacity', 1);
        $('.self').css('opacity', 1);
        $('.regis').css('opacity', 1);
        $('.regis-bottom').css('opacity', 1);
        $('.rayquaza').css('opacity', 1);

        var p = '';
        var o = '';
        for (var i in data.regis) {
            if (data.regis[i] == "purple") {
                p += '&#128995;';
            } else if (data.regis[i] == "orange") {
                o += '&#128992;';
            }
        }

        $('.purplescore').html(`${data.purple.value} <span>${p}</span>`);
        $('.orangescore').html(`${data.orange.value} <span>${o}</span>`);
        $('.selfscore').html(data.self.value);

        $('.purplekos').html(data.purple.kos);
        $('.orangekos').html(data.orange.kos);
    } else {
        clear();
    }

    $('.stacks').html(data.stacks);

    var elekis = {
        "none": ["none", "orange", "purple"],
        "purple": ["purple", "orange", "none"],
        "orange": ["orange", "purple", "none"],
    }

    for (var i in data.regis) {
        $(`.regis-${parseInt(i)+1} .regis-circle-${elekis[data.regis[i]][0]}`).css('opacity', 1);
        $(`.regis-${parseInt(i)+1} .regis-circle-${elekis[data.regis[i]][1]}`).css('opacity', 0);
        $(`.regis-${parseInt(i)+1} .regis-circle-${elekis[data.regis[i]][2]}`).css('opacity', 0);
    }

    for (var i = 0; i < data.bottom.length; i++) {
        $(`.regis-bottom-${i+1} .regis-bottom-circle-purple`).css('opacity', 0);
        $(`.regis-bottom-${i+1} .regis-bottom-circle-orange`).css('opacity', 0);
        $(`.regis-bottom-${i+1} .regis-bottom-circle-none`).css('opacity', 0);

        var obj = data.bottom[i];
        $(`.regis-bottom-${i+1} .regis-bottom-circle-${obj.team}`).css('opacity', 1);
        $(`.regis-bottom-img-${i+1}`).attr('src', `assets/img/${obj.name}.png`);
    }

    if (data.bottom.length < 3) {
        for (var i = data.bottom.length; i < 3; i++) {
            $(`.regis-bottom-${i+1} .regis-bottom-circle-purple`).css('opacity', 0);
            $(`.regis-bottom-${i+1} .regis-bottom-circle-orange`).css('opacity', 0);
            $(`.regis-bottom-${i+1} .regis-bottom-circle-none`).css('opacity', 1);

            $(`.regis-bottom-img-${i+1}`).attr('src', 'assets/img/objective.png');
        }
    }

    $(`.rayquaza-1 .rayquaza-circle-purple`).css('opacity', 0);
    $(`.rayquaza-1 .rayquaza-circle-orange`).css('opacity', 0);
    $(`.rayquaza-1 .rayquaza-circle-none`).css('opacity', 1);

    if (data.rayquaza) {
        $(`.rayquaza-1 .rayquaza-circle-none`).css('opacity', 0);
        $(`.rayquaza-1 .rayquaza-circle-${data.rayquaza}`).css('opacity', 1);
    }
}

function websocket() {
    let socket = new WebSocket(urlWS);
    socket.onmessage = function(event) {
        success(JSON.parse(event.data));
    };
    socket.onerror = error;
}

$(document).ready(() => {
    clear();

    // Test.
    if (false) {
        return setInterval(() => {
            success({
                "profile": "player",
                // "profile": "broadcaster",
                "version": version,
                "started": true,
                "seconds": 360,
                "purple": { "value": 195, "kos": 1 },
                "orange": { "value": 102, "kos": 1 },
                "self": { "value": 132, "kos": 1 },
                "stacks": 6,
                "regis": ["purple", "orange", "purple"],
                "bottom": [
                    { "name": "registeel", "team": "purple" },
                    { "name": "regirock", "team": "orange" },
                    { "name": "registeel", "team": "purple" },
                    { "name": "regice", "team": "orange" },
                ],
            });
        }, 1000);
    }

    const query = window.location.search;
    var args = query.split("?");

    if (args.length == 2 && args[1] == "http") {
        console.log(`[UniteHUD] creating http connection to ${urlHTTP}`);
        setInterval(http, 1000);
        return;
    }

    console.info(`[UniteHUD] creating websocket connection to ${urlWS} (add "?http" to connect to the http endpoint)`);

    setInterval(websocket, 1000);
});