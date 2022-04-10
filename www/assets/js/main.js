var loaders = [".", "..", "..."];
var index = 0;
var loggedError = false;

const drednaws = [
    // "7:00"
    7 * 60,
];

const bees = [
    //"8:50"
    (8 * 60) + 50,
    // "7:20"
    (7 * 60) + 20,
    // "5:50"
    (5 * 60) + 50,
    // "4:20"
    (4 * 60) + 20,
    // "2:50"
    (2 * 60) + 50,
    // "1:20"
    60 + 20,
];

function main() {
    $.ajax({
        type: 'GET',
        dataType: 'json',
        url: "http://localhost:17069/http",
        timeout: 1000,
        success: function(data, status) {
            $('.purple').html(data.purple.value);
            $('.orange').html(data.orange.value);
            $('.self').html(data.self.value);
            $('.error').html('');
            $('.banner').css('opacity', '0');

            loggedError = false;

            if (data.purple.value + data.orange.value + data.seconds + data.self.value + data.balls == 0) {
                $('.score').css('opacity', '0');
            } else {
                $('.score').css('opacity', '1');

                for (var i in drednaws) {
                    var dreadRotom = data.seconds - drednaws[i];
                    if (dreadRotom > 0) {
                        $('.rotom-seconds').css('opacity', '1');
                        $('.drednaw-seconds').css('opacity', '1');

                        $('.rotom-seconds .objective').html(dreadRotom);
                        $('.drednaw-seconds .objective').html(dreadRotom);

                        break;
                    } else {
                        $('.rotom-seconds').css('opacity', '0');
                        $('.drednaw-seconds').css('opacity', '0');
                    }
                }

                for (var i in bees) {
                    var until = data.seconds - bees[i];
                    if (until > 0) {
                        $('.vespiquen-seconds').css('opacity', '1');
                        $('.vespiquen-seconds .objective').html(until);

                        break;
                    } else {
                        $('.vespiquen-seconds').css('opacity', '0');
                    }
                }
            }
        },
        error: function(err) {
            $('.error').html(`Unite HUD reconnecting${loaders[index]}`);
            index = (index + 1) % loaders.length;
            $('.banner').css('opacity', '.5');

            $('.purple').html("");
            $('.orange').html("");
            // $('.seconds').html("");
            $('.self').html("");
            $('.score').css('opacity', '0');

            if (!loggedError) {
                console.error(err);
                loggedError = true
            }
        },
    });
};

$(document).ready(() => {
    setInterval(main, 1000);
});