[![Build Status](https://cloud.drone.io/api/badges/xbapps/xbvr/status.svg)](https://cloud.drone.io/xbapps/xbvr) ![GitHub release](https://img.shields.io/github/release/xbapps/xbvr.svg)
<br>
<sup><sub><em>Windows • macOS • Linux • Raspberry Pi</em></sub></sup>

<h1 align="center">
    <img src="https://i.imgur.com/T2UvcHc.png" width="250"/>
</h1>

<h3 align="center">
    The ultimate tool for managing your VR porn library.
</h3>

<p align="center">
    <strong>
        <a href="https://github.com/xbapps/xbvr/issues">Suggestions</a>
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
- Support for all the most popular VR sites: BadoinkVR, CzechVR Network, DDFNetworkVR, MilfVR, NaughtyAmericaVR, SexBabesVR, StasyQVR, TmwVRnet, VirtualRealPorn, VirtualTaboo, VRBangers, VRHush, VRLatina, WankzVR and many studios on SexLikeReal
- Directly supports DeoVR and HereSphere VR players via API
- Built-in DLNA streaming server compatible with popular VR players (Pigasus, Skybox, Mobile Station VR)
- Sleek and simple web UI
- Browse your content by cast, site, tags, and release date
- Available for Windows, macOS, Linux (including ARM builds for RaspberryPi)

## Download

The latest version is always available on the [releases page](https://github.com/xbapps/xbvr/releases).

App is also available in form of Docker image, which makes it possible to run in more specialized environments such as QNAP NAS - downloads at [GitHub Container Registry](https://github.com/xbapps/xbvr/pkgs/container/xbvr).

To run this container in docker:

```
docker run -t --name=xbvr --net=host --restart=always \
   --mount type=bind,source=/path/to/your/videos,target=/videos \
   --mount source=xbvr-config,target=/root/.config/ \
   ghcr.io/xbapps/xbvr:latest
```

Adding `-d` to the docker command will run the container in the background.

In docker, your videos will be mounted at /videos and you should add this path in Options -> Folders.

Please note that during the first run XBVR automatically installs `ffprobe` and `ffmpeg` codecs from [ffbinaries site](https://ffbinaries.com/downloads).

## Quick Start

Once launched, web UI is available at `http://127.0.0.1:9999`.

Before anything else, you must allow the app to scan sites and populate its scene metadata library. Click through to Options -> Scene Data and "Run scraper". This can take several minutes to complete. Wait for it to finish, and then go to Options -> Folders and add the folders where your video files are stored.

When it's all done, you should see your media not only in web UI, but also through DLNA server in your favourite VR player.

Enjoy!

## Questions & Suggestions

Ask your questions and suggest features on [Discord](https://discord.gg/wdCHXAG).

## Development

Make sure you have following installed:

- Go 1.21
- Node.js 12.x
- Yarn 1.17.x
- air (run `go install github.com/cosmtrek/air@latest` outside project directory)

Once all of the above is installed, running `yarn dev` from project directory launches file-watchers providing livereload for both Go and JavaScript.

## Development in Gitpod

This project is configured for use in Gitpod. It will provide you with a pre-built development environment with all the tools needed to compile XBVR.

When the workspace loads, `yarn dev` runs and it will build and start XBVR automatically. Every time you make a change to a file, watchers will automatically compile the relevant code.

Once XBVR is compiled and starts, a preview panel will open in the IDE. As you modify go files, the preview panel will reload with the latest changes. If you make changes to Vue, you'll need to reload the browser to load the updated JavaScript.

Currently, it's only possible to test XBVR core and Browser applications using Gitpod. Because DLNA requires a local network, you won't be able to connect to the DLNA server running in Gitpod. For most people, this is fine.

sqlite3 is included in the terminal. The XBVR database is located at /home/gitpod/.config/xbvr/main.db

sqlite-web is also included. To browse the db, you can run `sqlite_web /home/gitpod/.config/xbvr/main.db`.

Gitpod has GitHub integration and, once authorized, can fork this repo into your account, push/pull changes, and create pull requests.

Ready to get started?

[![Open in Gitpod](https://gitpod.io/button/open-in-gitpod.svg)](https://gitpod.io/#https://github.com/xbapps/xbvr)


### How To

#### Add specific filter to DeoVR
* On the XBVR scenes page, create a filter (cast, site, tags, etc.) and sort order, then create a "saved search" (see top left) and check "use as DeoVR list". 
* Inside DeoVR you will now see your saved search listed
#### Keyboard Shortcuts
* Global
? - Quick Find  
* Details Pane  
o - previous scene  
p - next scene  
e - edit scene  
w - toggle watchlist  
f - toggle favourite  
W - toggle Watched status (Capital W)  
g - toggles gallery / video window  
esc - closes details pane  
left arrow - cycles backwards in gallery / skips backwards in video  
right arrow - cycles forward in gallery / skips forward in video
* File Match Pane  
o - previous file  
p - next file  
left arrow - next page of search results  
right arrow - previous  page of search results  
esc - closes matching pane  
* Actor List  
o or left arrow - previous page of actors  
p or right arrow - next page of actors  
* Actor Details  
o - previous actor  
p - next actor  
left arrow - cycles backwards in gallery  
right arrow - cycles forward in gallery  
esc - closes details pane

#### using Command Line Arguments/Environment Variables
| Command line parameter | Environment Variable | Type | Description |
|------------------------|--------------|------|-------------|
| `--enableLocalStorage` | | boolean |Use local folder to store application data|
|	`--app_dir` | XBVR_APPDIR | String|path to the application directory|
|	`--cache_dir` | XBVR_CACHEDIR | String|path to the tempoarary scraper cache directory|
|	`--imgproxy_dir` | XBVR_IMAGEPROXYDIR | String|path to the imageproxy directory|
|	`--search_dir` | XBVR_SEARCHDIR | String| path to the Search Index directory|
|	`--preview_dir` | XBVR_VIDEOPREVIEWDIR | String| path to the Scraper Cache directory|
|	`--scriptsheatmap_dir` | XBVR_SCRIPTHEATMAPDIR | String| path to the scripts_heatmap directory|
|	`--myfiles_dir` | XBVR_MYFILESDIR | String| path to the myfiles directory for serving users own content (eg images|
|	`--databaseurl` | DATABASE_URL | String|override default database path|
|	`--web_port` | XBVR_WEB_PORT | Int| override default Web Page port 9999|
|	`--ws_addr` | XBVR_WS_ADDR | String| override default Websocket address from the default 0.0.0.0:9998|
|	`--db_connection_pool_size` | DB_CONNECTION_POOL_SIZE | Int| sets the connection pool size for mariadb databases|
|	`--concurrent_scrapers` | CONCURRENT_SCRAPERS | String| set the number of scrapers that run concurrently default 9999|
