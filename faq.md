# FAQ
In this page you can expect questions and answers on how to use, run, maintain, and other aspects of XBVR.
If you have a question that should be present in this FAQ, please share it in [Discord](https://discord.gg/wdCHXAG) and ask if it can be added to the FAQ and help improve this page.

## How to:

### Run XBVR
Well, I don't know how you missed it, but this information is on the main readme file. See the [Download](https://github.com/xbapps/xbvr#download) and [Quick Start](https://github.com/xbapps/xbvr#quick-start) sections carefully.

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
