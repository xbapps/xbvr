<script>
  import { Router, Route, navigate } from "svelte-routing";
  import Navbar from "./Navbar.svelte";
  import Scenes from "./scenes/index.svelte";
  import { Wampy } from "wampy";

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
    });
</script>

<Router url="{url}" basepath="/ui/">
  <Route path="scenes" component="{Scenes}" />

  <Navbar/>
</Router>
