# FAQ
In this page you can expect questions and answers on how to use, run, maintain, and other aspects of XBVR.
If you have a question that should be present in this FAQ, please share it in [Discord](https://discord.gg/wdCHXAG) and ask if it can be added to the FAQ and help improve this page.



## How to:

### Run XBVR
Well, I don't know how you missed it, but this information is on the main readme file. See the [Download](https://github.com/xbapps/xbvr#download) and [Quick Start](https://github.com/xbapps/xbvr#quick-start) sections carefully.

### Access XBVR in a VR headset
You can use your headset to access the XBVR library, if your device is on the same network, by using a browser or http-capable video player (DeoVR and Heresphere are examples) at this location:
```http://your-xbvr-server-ip:9999```

You must change _your-xbvr-server-ip_ to the IP address of the machine running XBVR. This depends on your network setup, search online if you don't know how to see your machine's IP address.

Alternatively, you can use the built-in DLNA functionality with DLNA-ready tools and access the VR videos. You won't have the full range of options available in the XBVR library, it will just be serving the files.

### Use DeoVR and Heresphere's http API functionalities
If you follow the above procedure using one of these two video players, they may redirect you to the API mode automatically. Instead of the XBVR's web view you'll see a VR focused user interface. If this does not happen, or if you want to bookmark it directly, use these direct calls:

```http://your-xbvr-server-ip:9999/deovr```

```http://your-xbvr-server-ip:9999/heresphere```

Make sure these views are **enabled** in the _Options_ section of XBVR.

### Add a filter to DeoVR
* On the XBVR scenes page, create a filter (cast, site, tags, etc.) and sort order, then create a "saved search" (see top left) and check "use as DeoVR list". 
* Inside DeoVR you will now see your saved search listed

### Migrate from sqlite to mariaDB
You need first to know the basics of deploying and working with MariaDB, there's plenty of tutorials online for you to see. After you have a working MariaDB setup, these are the XBVR specific points you need to do:
- Create the user and database in MariaDB for XBVR
- Export / import settings and other info to a JSON file

A Docker XBVR run with a MariaDB database will be something like this:

```docker run -d  --name=xbvr2 --network=host -e "DATABASE_URL=mysql://user:password@db_ip:3306/xbvr?charset=utf8mb4&parseTime=True&loc=Local"  --restart=always -v /path_on_your_nas:/videos:ro  -v xbvr-config:/root/.config/  ghcr.io/xbapps/xbvr:latest```

Adapt it according to your needs.



## Can I:

### Install XBVR in my headset applications?
No, XBVR works as a server in a Windows, macOS or Linux PC (armv7 and arm64 builds included). See above **Access XBVR in a VR headset**.

### Use a VR video player in my VR headset?
Besides using a browser, the DeoVR and Heresphere video players use a http API to create a VR-focused UI with many features.

### Add my 2D video collection to XBVR?
You can, but XBVR is made to be a VR video library. You might want to take a look at [stash](https://github.com/stashapp/stash) and [stash-vr](https://github.com/o-fl0w/stash-vr) for your 2D video collection, and leave the VR videos to XBVR.

### Connect a device like the Launch or Handy to use funscripts?
You can if this is supported by your video player.

### Add cuepoints on videos?
You can, and it's quite a powerful way to enjoy more your videos as you're able to get to the parts you want quickly.



## Help:

### XBVR is not working after a update
The first run after a update can some time depending on how large your collection is, be patient.



## Keyboard Shortcuts:
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
