<template>

</template>

<script>
import { Wampy } from 'wampy'

export default {
  name: 'Socket',
  data () {
    return {
      wsStatus: ''
    }
  },
  mounted () {
    const ws = new Wampy('/ws/', {
      realm: 'default',
      onConnect: () => {
        this.wsStatus = 'connected'
      },
      onClose: () => {
        this.wsStatus = 'disconnected'
      },
      onError: () => {
        this.wsStatus = 'disconnected'
      },
      onReconnect: () => {
        this.wsStatus = 'connecting'
      },
      onReconnectSuccess: () => {
        this.wsStatus = 'connected'
      }
    })

    ws.subscribe('service.log', (dataArr, dataObj) => {
      if (dataArr.argsDict.level == 'debug') {
        console.debug(dataArr.argsDict.message)
      }
      if (dataArr.argsDict.level == 'info') {
        console.info(dataArr.argsDict.message)
      }
      if (dataArr.argsDict.level == 'error') {
        console.error(dataArr.argsDict.message)
      }

      if (dataArr.argsDict.data.task === 'scrape') {
        this.$store.state.messages.lastScrapeMessage = dataArr.argsDict
      }

      if (dataArr.argsDict.data.task === 'scraperProgress') {
        if (dataArr.argsDict.message === 'DONE') {
          this.$store.state.messages.runningScrapers = []
        }

        if (dataArr.argsDict.data.started) {
          this.$store.state.messages.runningScrapers.push(dataArr.argsDict.data.scraperID)
        }

        if (dataArr.argsDict.data.completed) {
          this.$store.state.messages.runningScrapers.splice(this.$store.state.messages.runningScrapers.indexOf(dataArr.argsDict.data.scraperID), 1)
        }
      }

      if (dataArr.argsDict.data.task === 'rescan') {
        this.$store.state.messages.lastRescanMessage = dataArr.argsDict
      }
    })

    ws.subscribe('lock.change', (dataArr, dataObj) => {
      if (dataArr.argsDict.name === 'scrape') {
        this.$store.state.messages.lockScrape = dataArr.argsDict.locked
      }
      if (dataArr.argsDict.name === 'rescan') {
        this.$store.state.messages.lockRescan = dataArr.argsDict.locked
      }
    })

    ws.subscribe('state.change.optionsStorage', (arr, obj) => {
      this.$store.dispatch('optionsStorage/load')
    })

    ws.subscribe('options.previews.previewReady', (arr, obj) => {
      this.$store.commit('optionsPreviews/showPreview', { previewFn: arr.argsDict.previewFn })
    })

    // Remote
    ws.subscribe('remote.state', (arr, obj) => {
      this.$store.dispatch('remote/processMessage', arr.argsDict)
    })
  }
}
</script>

<style scoped>

</style>
