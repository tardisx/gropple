# gropple

A web service and bookmarklet to download videos with a single click.

![Screencast](/screencast.gif)

## Pre-requisites

* a passing familiarity with the command line
* youtube-dl (plus any of its required dependencies, like ffmpeg)
* golang compiler (if you'd like to build from source)

## Build

    go build

## Binaries

Binaries are available at https://github.com/tardisx/gropple/releases

## Running

    gropple -port 6283 -address http://hostname:6283 -path /downloads

With no arguments, it will listen on port 6283 and use an address of 'http://localhost:6283'.

The address must be specified so that the bookmarklet can refer to the correct
host if it is not running on your local machine. You may also need to specify
a different address if you are running it behind a proxy server or similar.

## Using

Bring up `http://localhost:6283` (or your chosen address) in your browser. You 
should see a link to the bookmarklet at the top of the screen, and the list of
downloads (currently empty).

Drag the bookmarklet to your favourites bar, or otherwise bookmark it as you 
see fit.

Whenever you are on a page with a video you would like to download, simply 
click the bookmarklet.

A popup window will appear, the download will start on the your gropple server 
and the status will be shown in the window.

You may close this window at any time without stopping the download, the status 
of all downloads is available on the index page.

## Using an alternative downloader

The default downloader is youtube-dl. It is possible to use a different downloader 
via the `-dl-cmd` command line option.

While `gropple` will use your `PATH` to find the executable, you may also want 
to specify a full path instead.o

So, for instance, to use `youtube-dlc` instead of `youtube-dl` and specify the 
full path:

`gropple -dl-cmd /home/username/bin/youtube-dlc`

Note that this is only the path to the executable. If you need to change the 
command arguments, see below.

## Changing the youtube-dl arguments

The default arguments passed to `youtube-dl` are:

* `--newline` (needed to allow gropple to properly parse the output)
* `--write-info-json` (optional, but provides information on the download in the corresponding .json file)
* `-f` and `bestvideo[ext=mp4]+bestaudio[ext=m4a]/best[ext=mp4]/best` (choose the type of video `youtube-dl` will download)

These are customisable on the command line for `gropple`. For example, to duplicate these default options, you would 
do:

`gropple -dl-args '--newline' -dl-args '--write-info-json' -dl-args '-f' -dl-args 'bestvideo[ext=mp4]+bestaudio[ext=m4a]/best[ext=mp4]/best`

## TODO

Many things. Please raise an issue after checking the [currently open issues](https://github.com/tardisx/gropple/issues).


