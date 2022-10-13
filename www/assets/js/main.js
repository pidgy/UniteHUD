const urlWS = "ws://127.0.0.1:17069/ws";
const urlHTTP = "http://127.0.0.1:17069/http";

var loaders = [".", "..", "..."];
var index = 0;
var loggedError = false;

var lastShake = 0;

function error(err) {
    $('.error').html(`Connecting${loaders[index]}`);
    index = (index + 1) % loaders.length;

    $('.twitter').css('opacity', '0');
    $('.logo').css('opacity', '.5');

    $('.purplescore').html("");
    $('.orangescore').html("");
    $('.selfscore').html("");

    $('.purple').css('opacity', 0);
    $('.orange').css('opacity', 0);
    $('.self').css('opacity', 0);
    $('.regis').css('opacity', 0);

    if (!loggedError) {
        loggedError = true

        console.error(`${err}`);

        setTimeout(() => {
            loggedError = false;
        }, 3600000);
    }

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
        error: function(err) {
            error(`[UniteHUD] failed to connect to server at ${urlHTTP} (${err})`);
        },
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

    if (data.version != "v1.0beta") {
        $('.error').html(`${data.version} Client required`);
        return shake();
    }

    if (!data.started) {
        $('.purple').css('opacity', 0);
        $('.orange').css('opacity', 0);
        $('.self').css('opacity', 0);
        $('.regis').css('opacity', 0);
        $('.logo').css('opacity', '.5');
        $('.error').html(`${data.version}`);
        return shake();
    }

    $('.error').html(``);

    if (data.seconds > 0) {
        $('.purple').css('opacity', 1);
        $('.orange').css('opacity', 1);
        $('.self').css('opacity', 1);
        $('.regis').css('opacity', 1);

        $('.purplescore').html(data.purple.value);
        $('.orangescore').html(data.orange.value);
        $('.selfscore').html(data.self.value);

        if (data.seconds < 120) {
            $('.purple').css('left', '750px');
            $('.orange').css('left', '1028px');
        } else {
            $('.purple').css('left', '759px');
            $('.orange').css('left', '1020px');
        }
    } else {
        $('.purple').css('opacity', 0);
        $('.orange').css('opacity', 0);
        $('.self').css('opacity', 0);
        $('.regis').css('opacity', 0);
        $('.logo').css('opacity', '.5');
        $('.error').html(``);
    }

    $('.stacks').html("_ ".repeat(data.stacks));

    var purpleregis = "";
    var orangeregis = "";
    var reg = "_ ";

    for (var i in data.regis) {
        var teams = ["none", "orange", "purple"];

        switch (data.regis[i]) {
            case "purple":
                teams = ["purple", "orange", "none"];
                purpleregis += reg
                break;
            case "orange":
                teams = ["orange", "purple", "none"];
                orangeregis += reg
                break;
        }

        $(`.regis-${i+1} .regis-circle-${teams[0]}`).css('opacity', 1);
        $(`.regis-${i+1} .regis-circle-${teams[1]}`).css('opacity', 0);
        $(`.regis-${i+1} .regis-circle-${teams[2]}`).css('opacity', 0);
    }

    switch (occurences(orangeregis, "_")) {
        case 0:
        case 1:
            $('.orangeregis-container').css("left", "1115px");
            break;
        case 2:
            $('.orangeregis-container').css("left", "1090px");
            break;
        case 3:
            $('.orangeregis-container').css("left", "1065px");
            break;
    }

    $(`.orangeregis`).html(orangeregis);
    $(`.purpleregis`).html(purpleregis);
}

function websocket() {
    let socket = new WebSocket(urlWS);
    socket.onopen = function(e) {}
    socket.onmessage = function(event) {
        success(JSON.parse(event.data));
    };
    socket.onerror = function(err) {
        error(`[UniteHUD] failed to connect to server at ${urlWS} (${JSON.stringify(err)})`);
    };
}

$(document).ready(() => {
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