var loaders = [".", "..", "..."];
var index = 0;
var loggedError = false;

function score() {
    $.ajax({
        type: 'GET',
        dataType: 'json',
        url: "http://localhost:17069/http",
        timeout: 1000,
        success: function(data, status) {
            $('.purple').html(data.purple.value);
            $('.orange').html(data.orange.value);
            $('.self').html(data.self.value);
            $('.seconds').html(data.seconds);
            $('.error').html('');

            loggedError = false;

            if (data.purple.value + data.orange.value + data.seconds + data.self.value + data.balls == 0) {
                $('.score').css('opacity', '0');
            } else {
                $('.score').css('opacity', '1');
            }
        },
        error: function(err) {
            $('.error').html(`Unite HUD reconnecting${loaders[index]}`);
            index = (index + 1) % loaders.length;

            $('.purple').html("");
            $('.orange').html("");
            $('.seconds').html("");
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
    setInterval(score, 1000);
});