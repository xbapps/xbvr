<script>
  import { Router, Route, navigate } from "svelte-routing";
  import Navbar from "./Navbar.svelte";
  import Scenes from "./scenes/index.svelte";
  import Options from "./options/index.svelte";
  import { Wampy } from "wampy";
  import { lockRescan, lastRescanMessage, lockScrape, lastScrapeMessage } from "./store/log.js";

  let url = "";
  let wsStatus = "";

  let ws = new Wampy("/ws/", {
    realm: "default",
    onConnect: () => {
      wsStatus = "connected";
      console.log("connected");
    },
    onClose: () => {
      wsStatus = "disconnected";
    },
    onError: () => {
      wsStatus = "disconnected";
    },
    onReconnect: () => {
      wsStatus = "connecting";
    },
    onReconnectSuccess: () => {
      wsStatus = "connected";
    }
  });

  ws
    .subscribe("service.log", (dataArr, dataObj) => {
      if (dataArr.argsDict.level == "debug") {
        console.debug(dataArr.argsDict.message);
      }
      if (dataArr.argsDict.level == "info") {
        console.info(dataArr.argsDict.message);
      }
      if (dataArr.argsDict.level == "error") {
        console.error(dataArr.argsDict.message);
      }

      if (dataArr.argsDict.data.task === "scrape") {
        $lastScrapeMessage = dataArr.argsDict;
      }

      if (dataArr.argsDict.data.task === "rescan") {
        $lastRescanMessage = dataArr.argsDict;
      }
    });

  ws
    .subscribe("lock.change", (dataArr, dataObj) => {
      if (dataArr.argsDict.name === "scrape") {
        $lockScrape = dataArr.argsDict.locked;
      }
      if (dataArr.argsDict.name === "rescan") {
        $lockRescan = dataArr.argsDict.locked;
      }
    })
</script>

<Router url="{url}" basepath="/ui/">
  <Route path="/" component="{Scenes}" />
  <Route path="/options" component="{Options}" />
  <Navbar/>
</Router>
