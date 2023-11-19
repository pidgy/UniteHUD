const { BrowserWindow } = require('electron')
const fs = require('fs')
const path = require('path')

// Offscreen BrowserWindow
let offscreenWindow
let nativeImage
    // Exported readItem function
module.exports = (url, callback) => {
    // Create offscreen window
    offscreenWindow = new BrowserWindow({
        width: 500,
        height: 500,
        show: false,
        webPreferences: {
            offscreen: true
        }
    })

    // Load item url
    offscreenWindow.loadURL(url)
    console.log("readitem")

    // Wait for content to finish loading
    offscreenWindow.webContents.on('did-stop-loading', async() => {

        // Get page title
        let title = offscreenWindow.getTitle()
        console.log(title)

        // Get screenshot (thumbnail)
        nativeImage = await offscreenWindow.webContents.capturePage()
            .then(image => {
                fs.writeFileSync('test.png', image.toPNG(), (err) => {
                    if (err) throw err
                })
                console.log('It\'s saved!')
                return image.toDataURL();
            });

        let obj = {
            title: title,
            url: url,
            image: nativeImage
        }
        callback(obj)

        offscreenWindow.close()
        offscreenWindow = null
    })
}