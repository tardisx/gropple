# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]

## [v0.6.0] - 2022-04

- Configurable destinations for downloads
- When downloading from a playlist, show the total number of videos and how many have been downloaded
- Show version in web UI
- Improve index page (show URL of queued downloads instead of nothing)
- Add docker support
- Fixes and improvements to capturing output info and showing it in the UI
- Fixes to handling of queued downloads

## [v0.5.5] - 2022-04-09

- Fix a bug which would erase configuration when migrating from v1 to v2 config

## [v0.5.4] - 2022-04-07

- Check the chosen command exists when configuring a profile
- Improve documentation
- Add a stop button in the popup to abort a download (Linux/Mac only)
- Move included JS to local app instead of accessing from a CDN
- Make the simultaneous download limit apply to each unique domain
- Support "portable" mode, reading gropple.yml from the current directory, if present

## [v0.5.3] - 2021-11-21

- Add config option to limit number of simultaneous downloads
- Remove old download entries from the index after they are complete

## [v0.5.2] - 2021-10-26

- Provide link to re-display the popup window from the index
- Visual improvements

## [v0.5.1] - 2021-10-25

- Add note about adblockers potentially blocking the popup
- Make it possible to refresh the popup window without initiating a new download

## [v0.5.0] - 2021-10-01

- No more command line options, all configuration is now app-managed
- Beautiful (ok, less ugly) new web interface
- Multiple youtube-dl profiles, a profile can be chosen for each download
- Bundled profiles include a standard video download and an mp3 download
- Configuration via web interface, including download profile configuration

## [v0.4.0] - 2021-09-26

- Moved to semantic versioning
- Automatic version check, prompts for upgrade in GUI
- Fixed regex to properly match "merging" lines
- Automatically refresh index page

## [0.03] - 2021-09-24

- Add option to change command (to use youtube-dlc or other forks) and command line arguments
- Improve log display in popup
- Improve documentation (slightly)

## [0.02] - 2021-09-22

- Fix #4 so that deleted files are removed from the results

## [0.01] - 2021-09-22

- Initial release
