const log = (msg, level = console.log) => {
    level(`[UniteHUD Projector] ${msg}`);
};

// namespace MJPEG { ...
var MJPEG = ((module) => {
    "use strict";

    // class Stream { ...
    module.Stream = function(args) {
        var self = this;
        var autoStart = args.autoStart || false;

        self.url = args.url;
        self.refreshRate = args.refreshRate;
        self.onStart = args.onStart || null;
        self.onFrame = args.onFrame || null;
        self.onStop = args.onStop || null;
        self.callbacks = {};

        self.img = new Image();
        if (autoStart) {
            self.img.onload = self.start;
        }
        self.img.src = self.url;

        function setRunning(running) {
            self.running = running;

            if (!self.running) {
                clearInterval(self.animationFrame);

                self.img.src = `url("../splash/projector.png")`;

                if (self.onStop) {
                    self.onStop();
                }

                return;
            }

            self.img.src = self.url;

            self.animationFrame = setInterval(
                () => {
                    self.onFrame(self.img);
                },
                self.refreshRate,
            );

            if (self.onStart) {
                self.onStart();
            }
        }

        self.start = () => { setRunning(true); };
        self.stop = () => { setRunning(false); };
    };

    // class Render { ...
    module.Render = function() {
        log(`Rendering`);
        var self = this;

        self.stream = new module.Stream({
            url: "http://localhost:17069/stream",
            refreshRate: 1,
            onFrame: (img) => {
                if (img.height === 0) {
                    return;
                }
                context.drawImage(img, 0, 0);
            },
            onStart: () => { log("[Media Player] Started"); },
            onStop: () => { log("[Media Player] Stopped"); },
        });

        var canvas = document.getElementById('projector-canvas');
        var context = canvas.getContext("2d");

        self.stream.start();
    };

    return module;

})(MJPEG || {});

$(document).ready(() => {
    document.addEventListener('astilectron-ready', new MJPEG.Render);
    document.addEventListener('keyup', (e) => {
        if (e.key != "Escape") {
            return;
        }

        if (typeof astilectron === undefined) {
            log(`Ignoring key event, incompatible environment`, console.warn);
            return;
        }

        astilectron.sendMessage("close");
    });
});