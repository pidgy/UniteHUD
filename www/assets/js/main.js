const version = "v1.0beta";
const urlWS = "ws://127.0.0.1:17069/ws";
const urlHTTP = "http://127.0.0.1:17069/http";

var loaders = [".", "..", "..."];
var index = 0;
var loggedError = false;

var lastShake = 0;

function clear() {
    $('.purple').css('opacity', 0);
    $('.orange').css('opacity', 0);
    $('.self').css('opacity', 0);
    $('.regis').css('opacity', 0);
    $('.error').css('opacity', '.9');
}

function error(err) {
    clear();

    if (typeof err === "string") {
        $('.error').html(`${err}`);

        if (!loggedError) {
            loggedError = true

            console.error(`${err}`);

            setTimeout(() => {
                loggedError = false;
            }, 3600000);
        }
    } else {
        $('.error').html(`Connecting${loaders[index]}`);
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

    if (data.version != version) {
        error(`${data.version} client required`);
        return shake();
    }

    if (!data.started) {
        clear();
        $('.error').html(`${version}`);
        return shake();
    }

    if (data.seconds > 0) {
        $('.purple').css('opacity', 1);
        $('.orange').css('opacity', 1);
        $('.self').css('opacity', 1);
        $('.regis').css('opacity', 1);

        $('.purplescore').html(data.purple.value);
        $('.orangescore').html(data.orange.value);
        $('.selfscore').html(data.self.value);
    } else {
        clear();
        $('.error').html(``);
    }

    $('.stacks').html(data.stacks);

    var cache = {
        "none": ["none", "orange", "purple"],
        "purple": ["purple", "orange", "none"],
        "orange": ["orange", "purple", "none"],
    }

    for (var i in data.regis) {
        $(`.regis-${parseInt(i)+1} .regis-circle-${cache[data.regis[i]][0]}`).css('opacity', 1);
        $(`.regis-${parseInt(i)+1} .regis-circle-${cache[data.regis[i]][1]}`).css('opacity', 0);
        $(`.regis-${parseInt(i)+1} .regis-circle-${cache[data.regis[i]][2]}`).css('opacity', 0);
    }
}

function websocket() {
    let socket = new WebSocket(urlWS);
    socket.onmessage = function(event) {
        success(JSON.parse(event.data));
    };
    socket.onerror = error;
}

const test = false;

$(document).ready(() => {
    clear();

    if (test) {
        $('.error').html(``);

        return success({
            "regis": ["purple", "orange", "none"],
            "version": version,
            "started": true,
            "seconds": 360,
            "purple": { "value": 195 },
            "orange": { "value": 102 },
            "self": { "value": 132 },
            "stacks": 6,
        });
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