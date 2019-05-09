# XBVR

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

## Quick start

NOTE: all the below will become obsolete at some point as the ultimate goal is to move everything into web-based UI, so no command line interactions would be neccesary.

To get started, you'll need to add some *volumes* (folders) first, you can do that via command line interface:

    xbvr volumes add <PATH_TO_FOLDER>

Let's scrape some metadata (it takes some time!):

    xbvr task scrape

Now let's scan and match data to files:

    xbvr volume rescan

After all of this, you can finally run the app:

    xbvr run

Once launched, web UI is available at `http://127.0.0.1:9999` and you should be able to see new DLNA source in your VR player of choice.
