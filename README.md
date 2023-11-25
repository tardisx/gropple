# gropple

A frontend to youtube-dl (or compatible forks, like yt-dlp) to download videos
with a single click, straight from your web browser.

![Screencast](/screencast.gif)

## Installing

### From Source

    go build

### Standalone Binaries

Binaries are available at <https://github.com/tardisx/gropple/releases> for most
platforms.

## Running

### From Binaries

     ./gropple

There are no command line arguments. All configuration is done via the web
interface. The address will be printed after startup:

    2023/11/22 22:42:06 Starting gropple v1.0.0 - https://github.com/tardisx/gropple
    2023/11/22 22:42:07 Configuration loaded from /Users/username/path/config.yml
    2023/11/22 22:42:07 Visit http://localhost:6123 for details on installing the bookmarklet and to check status

### Docker

Copy the `docker-compose.yml` to a directory somewhere.

Edit the two `volume` entries to point to local paths where you would like to
store the config file, and the downloads (the path on the left hand side of the
colon).

Run `docker-compose up -d` to start the program.

Note that the docker images include `yt-dlp` and `ffmpeg` and are thus
completely self-contained.

Run `docker-compose logs` to see the output of the program, if you are having
problems.

## Using

Bring up `http://localhost:6283` (or the appropriate host if you are running it
on a different machine) in your browser. You should see a link to the
bookmarklet at the top of the screen, and the list of downloads (currently
empty).

Drag the bookmarklet to your favourites bar, or otherwise bookmark it as you see
fit. Any kind of browser bookmark should work. The bookmarklet contains embedded
javascript to pass the URL of whatever page you are currently on back to
gropple.

Whenever you are on a page with a video you would like to download just click
the bookmarklet.

A popup window will appear. Choose a download profile and the download will
start. The status will be shown in the window, updating in real time.

There is also an optional "download option" you can choose. These are discussed
below.

You may close this window at any time without stopping the download, the status
of all downloads is available on the index page. Clicking on the id number will
show the popup again.

## Configuration

Click the "config" link on the index page to configure gropple.

The options in each part are dicussed below.

### Server

#### Port and Server Address

You can configure the port number here if you do not want the default of `6123`.

If you are running it on a machine other than `localhost` you will need to set
the "server address" to ensure the bookmarklet has the correct URL in it.
Similarly, if you are running it behind a reverse proxy, the address here must
match what you would type in the browser so that the bookmarklet will work
correctly.

#### Download path

The download path specifies where downloads will end up, *if* no specific `-o`
options are passed to `yt-dlp`.

#### Maximum active downloads per domain

Gropple will limit the number of downloads per domain to this number. Increasing
this will likely result in failed downloads when server rate limiters notice
you.

#### UI popup size

Changes the size of the popup window.

### Download Profiles

Gropple's default configuration uses `yt-dlp` and has two profiles set up, one
for downloading video, the other for downloading audio (mp3).

Each download profile consists of a name (for your reference), a command to run,
and a number of arguments.

Note that gropple does not include any downloaders, you have to install them
separately (unless using the docker image).

If you would like to use a youtube-dl compatible fork or change the options you
can do so here. Create as many profiles as you wish, whenever you start a
download you can choose the appropriate profile.

Note that the command arguments must each be specified separately - see the
default configuration. For example, if you have a single argument like
`--audio-format mp3`, it will be parsed by the `yt-dlp` as a single, long
unknown argument, and will fail. This needs to be configured as two arguments,
`--audio-format` and `mp3`.

While gropple will use your `PATH` to find the executable, you can also specify
a full path instead. Note that any tools that the downloader calls itself (for
instance, `ffmpeg`) will need to be available on your path.

### Download Options

There are also an arbitrary amount of Download Options you can configure. Each
one specifies one or more extra arguments to add to the downloader command line.
The most common use for this is to have customised download paths. For instance,
sometimes you might want to bundle all files into a single directory, other
times you might want to separate files by download playlist URL or similar.

Most of this is done directly through appropriate options for `yt-dlp`, see the
[output template
documentation](https://github.com/yt-dlp/yt-dlp#output-template).

However, gropple offers two extra substitutions:

  * `%GROPPLE_HOST%`
  * `%GROPPLE_PATH%`

These will be replaced with the hostname, and path of the download,
respectively.

So, a playlist URL `https://www.youtube.com/@UsernameHere`

With a download option setup like this:

    * Name of Option: Split by Host and Path
    * Arguments:
      * -o
      * /Downloads/%GROPPLE_HOST%/%GROPPLE_PATH%/%(title)s [%(id)s].%(ext)s

Will result in downloads going into the path
`/Downloads/www.youtube.com/@UsernameHere/...`.

Note that this also means that `yt-dlp` can resume partially downloaded files, and
also automatically 'backfill', downloading only files that have not been
downloaded yet from that playlist.

## Downloading a list of URL's in bulk

From main index page you can click the "Bulk" link in the menu to bring up the
bulk queue page.

In all respects this acts the same as the usual bookmarklet, but it has a
textbox for pasting many URLs at once. All downloads will be queued immediately.

## Portable mode

If you'd like to use gropple from a USB stick or similar, copy the config file
from its default location (shown when you start gropple) to the same location as
the binary, and rename it to `gropple.yml`.

## Problems

Many download problems are diagnosable via the log - check in the popup window
and scroll the log down to the bottom. The most common problem is that `yt-dlp`
cannot be found, or its dependency (like `ffmpeg`) cannot be found on your path.

Gropple only calls external tools like `yt-dlp` to do the downloading. If you
are having problems downloading from a site, make sure that `yt-dlp` is updated
to the latest version (`yd-dlp -U`).

For other problems, please file an issue on github.

## TODO

Many things. Please raise an issue after checking the [currently open
issues](https://github.com/tardisx/gropple/issues).
