const state = {
  lockScrape: false,
  lastScrapeMessage: '',
  lockRescan: false,
  lockPreview: false,
  lastRescanMessage: '',
  lastProgressMessage: '',
  runningScrapers: []
}

export default {
  namespaced: true,
  state
}
