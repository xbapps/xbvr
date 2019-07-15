<template>

</template>

<script>
  import { Wampy } from "wampy";

  export default {
    name: "Socket",
    data() {
      return {
        wsStatus: "",
      }
    },
    mounted() {
      let ws = new Wampy("/ws/", {
        realm: "default",
        onConnect: () => {
          this.wsStatus = "connected";
        },
        onClose: () => {
          this.wsStatus = "disconnected";
        },
        onError: () => {
          this.wsStatus = "disconnected";
        },
        onReconnect: () => {
          this.wsStatus = "connecting";
        },
        onReconnectSuccess: () => {
          this.wsStatus = "connected";
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
            this.$store.state.messages.lastScrapeMessage = dataArr.argsDict;
          }

          if (dataArr.argsDict.data.task === "rescan") {
            this.$store.state.messages.lastRescanMessage = dataArr.argsDict;
          }
        });

      ws
        .subscribe("lock.change", (dataArr, dataObj) => {
          if (dataArr.argsDict.name === "scrape") {
            this.$store.state.messages.lockScrape = dataArr.argsDict.locked;
          }
          if (dataArr.argsDict.name === "rescan") {
            this.$store.state.messages.lockRescan = dataArr.argsDict.locked;
          }
        })
    }
  }
</script>

<style scoped>

</style>