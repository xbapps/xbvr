#!/bin/bash
# this script is used to start xbvr in docker environments
# it copies release files from /tmp in the docker image to the exported app directory
# and then startes xbvr

TARGET_DIR="/root/.config/xbvr/xbvr_data/migrations/release"
SOURCE_DIR="/tmp/xbvr_data"

# Check if target directory exists
if [ ! -d "$TARGET_DIR" ]; then
    mkdir -p "$TARGET_DIR"
    chmod 777 "$TARGET_DIR"
fi

TARGET_DIR="/root/.config/xbvr/xbvr_data/migrations/custom"
if [ ! -d "$TARGET_DIR" ]; then
    mkdir -p "$TARGET_DIR"
    chmod 777 "$TARGET_DIR"
fi

# Copy recursively from /tmp/data to the target directory
cp -ru /tmp/xbvr_data /root/.config/xbvr

/usr/bin/xbvr