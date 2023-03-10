(function() {
    var videos = document.querySelectorAll("video");
    if (videos.length > 1) {
        var video = videos[0];
        video.height = 1080;
        video.width = 1920;
        video.requestFullscreen();
        /*
        for (var i = 1; i < videos.length; i++) {
            if (!video.videoWidth) {
                video = videos[i];
            } else if (videos[i].videoWidth && (videos[i].videoWidth > video.videoWidth)) {
                video = videos[i];
            }
        }
*/
        //document.body.appendChild(video);
    } else if (videos.length) {
        document.body.appendChild(videos[0]);
    }
})();