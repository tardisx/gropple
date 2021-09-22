# gropple

A web service and bookmarklet to download videos with a single click.

<div style='position:relative; padding-bottom:calc(46.95% + 44px)'><iframe src='https://gfycat.com/ifr/AppropriateHeftyHartebeest' frameborder='0' scrolling='no' width='100%' height='100%' style='position:absolute;top:0;left:0;' allowfullscreen></iframe></div>

## Pre-requisites

* a passing familiarity with the command line
* youtube-dl (plus any of its required dependencies, like ffmpeg)
* golang compiler

## Build

    go build

## Binaries

Binaries are available at https://github.com/tardisx/gropple/releases

## Running

    gropple -port 6283 -address http://hostname:6283 -path /downloads

With no arguments, it will listen on port 6283 and use an address of 'http://localhost:6283'.

The address must be specified so that the bookmarklet can refer to the correct
host when it is not running on your local machine. You may also need to specify
a different address if you are running it behind a proxy server or similar.

## Using

Bring up `http://localhost:6283` (or your chosen address) in your browser. You should see a link to the bookmarklet at the top of the screen, and the list of downloads (currently empty).

Drag the bookmarklet to your favourites bar, or otherwise bookmark it as you see fit.

Whenever you are on a page with a video you would like to download, simply click the bookmarklet.

A popup window will appear, the download will start on the your gropple server and the status will be shown in the window.

You may close this window at any time without stopping the download, the status of all downloads is available on the index page.

## TODO

Many things. Please raise an issue after checking the [currently open issues](https://github.com/tardisx/gropple/issues).


