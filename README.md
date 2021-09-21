# gropple

## Pre-requisites

* youtube-dl (plus any of its required dependencies, like ffmpeg)
* golang compiler

## Build

    go build

## Running

    ./gropple -port 8000 -address http://hostname:8000 -path /downloads

With no arguments, it will listen on port 8000 and use an address of 'http://localhost:8000'.

The address must be specified so that the bookmarklet can refer to the correct
host when it is not running on your local machine. You may also need to specify
a different address if you are running it behind a proxy server or similar.

## Using


