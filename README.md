# XBVR

[![Build Status](https://cloud.drone.io/api/badges/cld9x/xbvr/status.svg)](https://cloud.drone.io/cld9x/xbvr) ![GitHub release](https://img.shields.io/github/release/cld9x/xbvr.svg)

<img src="https://i.imgur.com/Q3UdJhV.jpg" width="500"/>

All-in-one tool for your VR porn library.

## Features

- scan multiple folders for content
- built-in scrapers for popular VR sites - including filenames of downloadable media, so your files could be automatically matched with metadata
- built-in DLNA streaming server compatible with popular VR players (Pigasus, Skybox, Mobile Station VR)
- browse your content by cast, site, tags, release date using web UI (taxonomy also reflected in DLNA)
- available for Windows, macOS, Linux (including ARM builds for RaspberryPi)

## Download

Latest version is always available at [releases page](https://github.com/cld9x/xbvr/releases).

Please note that during first run XBVR automatically downloads `ffprobe` and `ffmpeg` binaries [ffbinaries site](https://ffbinaries.com/downloads).

## Quick start

Once launched, web UI is available at `http://127.0.0.1:9999`.

At first there will be nothing interesting, so click through to Options -> Scene Data and import bundled data. Wait for it to finish, and then go to Options -> Folders and add path to the folders where your media files are stored.

When it's all done, you should see your media not only in web UI, but also through DLNA server in your favourite VR player.

Enjoy!
