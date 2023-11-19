$(document).ready(() => {
    const stamp = "client";

    document.addEventListener('astilectron-ready', () => {
        astilectron.onMessage(function(message) {
            console.log(`received "${message}"`);

            if (message == "render::screenshot") {
                return `${stamp}::${message}`;
            }

            return `${stamp}::unknown::${message}`
        });
    });
});