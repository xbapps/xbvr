const state = {
  lockScrape: false,
  lastScrapeMessage: '',
  lockRescan: false,
  lastRescanMessage: '',
  lastProgressMessage: '',
  runningScrapers: []
}

export default {
  namespaced: true,
  state
}
