const CONTROL_LEVEL = {
    0: "None",
    1: "CONTROL_LEVEL_READ_OBS",
    2: "CONTROL_LEVEL_READ_USER",
    3: "CONTROL_LEVEL_BASIC",
    4: "CONTROL_LEVEL_ADVANCED",
    5: "CONTROL_LEVEL_ALL",

    NONE: 0,
    CONTROL_LEVEL_READ_OBS: 1,
    CONTROL_LEVEL_READ_USER: 2,
    CONTROL_LEVEL_BASIC: 3,
    CONTROL_LEVEL_ADVANCED: 4,
    CONTROL_LEVEL_ALL: 5
};

var control_level = CONTROL_LEVEL.NONE;

function log(msg) {
    console.error(`[UniteHUD] [OBS] ${msg}`);
}

window.addEventListener('obsSourceVisibleChanged', function(event) {
    log("obsstudio.obsSourceVisibleChanged: " + event);
});

window.addEventListener('obsRecordingStarting', function(event) {
    log("obsstudio.obsRecordingStarting: " + event);
});

window.addEventListener('obsSourceActiveChanged', function(event) {
    log("obsstudio.obsSourceActiveChanged: " + event);
});

$(document).ready(() => {
    if (!window.obsstudio) {
        return;
    }

    window.obsstudio.getControlLevel(function(level) {
        control_level = CONTROL_LEVEL[level];

        log("obsstudio.pluginVersion: " + window.obsstudio.pluginVersion);
        log("obsstudio.controlLevel:  " + CONTROL_LEVEL[level]);
    });
});