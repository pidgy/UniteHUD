var loaders = [".", "..", "..."];
var index = 0;

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

            if (data.purple.value + data.orange.value + data.seconds + data.self.value == 0) {
                $('.score').css('opacity', '0');
            } else {
                $('.score').css('opacity', '1');
            }

            if (data.purple.value > 99) {
                $('.purple').css("font-size", "30px");
            } else {
                $('.purple').css("font-size", "40px");
            }

            if (data.orange.value > 99) {
                $('.orange').css("font-size", "30px");
            } else {
                $('.orange').css("font-size", "40px");
            }
        },
        error: function(err) {
            $('.error').html(`Unite HUD reconnecting${loaders[index]}`);
            index = (index + 1) % loaders.length;

            $('.purple').html("");
            $('.orange').html("");
            $('.seconds').html("");
            $('.self').html("");
            $('.score').css('opacity', '1');
        },
    });
};

$(document).ready(() => {
    setInterval(score, 1000);
});