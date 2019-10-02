[![Build Status](https://cloud.drone.io/api/badges/cld9x/xbvr/status.svg)](https://cloud.drone.io/cld9x/xbvr) ![GitHub release](https://img.shields.io/github/release/cld9x/xbvr.svg)
<br>
<sup><sub><em>Windows 10 • macOS • Linux • Raspberry Pi</em></sub></sup>

<h1 align="center">
    <img src="https://i.imgur.com/T2UvcHc.png" width="250"/>
</h1>

<h3 align="center">
    The ultimate tool for managing your VR porn library.
</h3>

<p align="center">
    <strong>
        <a href="https://feedback.xbvr.app/">Suggestions</a>
        •
        <a href="https://discord.gg/wdCHXAG">Discord</a>
    </strong>
</p>

<p align="center" stlye="text-shadow: 2px 2px">
    <kbd><img src="https://i.imgur.com/Q3UdJhV.jpg" width="500"/></kbd>
    <br>
</p>


## Features

- Automatically match title, tags, cast, cover image, and more to your videos
- Support for all the most popular VR sites: BadoinkVR, CzechVR Network, DDFNetworkVR, MilfVR, NaughtyAmericaVR, SexBabesVR, StasyQVR, TmwVRnet, VirtualRealPorn, VirtualTaboo, VRBangers, VRHush, VRLatina and WankzVR
- Supports DeoVR web mode
- Built-in DLNA streaming server compatible with popular VR players (Pigasus, Skybox, Mobile Station VR)
- Sleek and simple web UI
- Browse your content by cast, site, tags, and release date
- Available for Windows, macOS, Linux (including ARM builds for RaspberryPi)

## Download

The latest version is always available on the [releases page](https://github.com/cld9x/xbvr/releases).

App is also available in form of Docker image, which makes it possible to run in more specialized environments such as QNAP NAS - downloads at [Docker Hub](https://hub.docker.com/r/cld9x/xbvr). 

Please note that during the first run XBVR automatically installs `ffprobe` and `ffmpeg` codecs from [ffbinaries site](https://ffbinaries.com/downloads).

## Quick Start

Once launched, web UI is available at `http://127.0.0.1:9999`.

Before anything else, you must allow the app to scan sites and populate its scene metadata library. Click through to Options -> Scene Data and "Run scraper". This can take several minutes to complete. Wait for it to finish, and then go to Options -> Folders and add the folders where your video files are stored.

When it's all done, you should see your media not only in web UI, but also through DLNA server in your favourite VR player.

Enjoy!

## Questions & Suggestions

Submit and vote on features at [Feedback site](https://feedback.xbvr.app/).

Ask your questions and suggest features on [Discord](https://discord.gg/wdCHXAG).

## Development

Make sure you have following installed:

- Go 1.12
- Node.js 10.x
- Yarn 1.17.x
- fileb0x (run `go get github.com/UnnoTed/fileb0x` outside project directory)
- modd (run `go get github.com/cortesi/modd/cmd/modd` outside project directory)

Once all of the above is installed, running `modd` from project directory launches file-watchers providing livereload for both Go and JavaScript.

## Development in Gitpod

This project is configured for use in Gitpod. It will provide you with a pre-built development environment with all the tools needed to compile XBVR.

When the workspace loads, modd runs and will build and start XBVR automatically. Every time you make a change to a file, modd will automatically compile the relevant code.

Once XBVR is compiled and starts, a preview panel will open in the IDE. As you modify files, the preview panel will reload with the latest changes.

Currently, it's only possible to test XBVR core and Browser applications using Gitpod. Because DLNA requires a local network, you won't be able to connect to the DLNA server running in Gitpod. For most people, this is fine.

sqlite3 is included in the terminal. The XBVR database is located at /home/gitpod/.config/xbvr/main.db

Gitpod has GitHub integration and, once authorized, can fork this repo into your account, push/pull changes, and create pull requests.

Ready to get started?

[![Open in Gitpod](https://gitpod.io/button/open-in-gitpod.svg)](https://gitpod.io/#https://github.com/cld9x/xbvr)
