const addCSS = css => document.head.appendChild(document.createElement("style")).innerHTML = css;

var styles = " \
.html5-video-player {\
    z-index:unset!important;\
}\
.html5-video-container {	\
    z-index:unset!important;\
}\
video { \
    width: 100vw!important;height: 100vh!important;  \
    left: 0px!important;    \
    object-fit: cover!important;\
    top: 0px!important;\
    overflow:hidden;\
    z-index: 2147483647!important;\
    position: fixed!important;\
}\
body {\
    overflow: hidden!important;\
}";

addCSS(styles)