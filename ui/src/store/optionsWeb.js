import ky from 'ky'

const state = {
  loading: false,
  web: {
    tagSort: 'By Tag Count',
    sceneHidden: true,
    sceneWatchlist: true,
    sceneFavourite: true,
    sceneWishlist: true,
    sceneWatched: false,
    sceneEdit: false,
    sceneDuration: false,
    sceneCuepoint: true,
    showHspFile: true,
    showSubtitlesFile: true,
    sceneTrailerlist: true,
    updateCheck: true,
    isAvailOpacity: 40,
    showScriptHeatmap: false,
    showAllHeatmaps: false,
    showOpenInNewWindow: true,
    sceneCardAspectRatio: "1:1",
    sceneCardScaleToFit: true,
    actorCardAspectRatio: "1:1",
    actorCardScaleToFit: true,
    updateCheck: true
  }
}

const mutations = {}

const actions = {
  async load ({ state }) {
    state.loading = true
    ky.get('/api/options/state')
      .json()
      .then(data => {
        state.web.tagSort = data.config.web.tagSort
        state.web.sceneHidden = data.config.web.sceneHidden
        state.web.sceneWatchlist = data.config.web.sceneWatchlist
        state.web.sceneFavourite = data.config.web.sceneFavourite
        state.web.sceneWishlist = data.config.web.sceneWishlist
        state.web.sceneWatched = data.config.web.sceneWatched
        state.web.sceneEdit = data.config.web.sceneEdit
        state.web.sceneDuration = data.config.web.sceneDuration
        state.web.sceneCuepoint = data.config.web.sceneCuepoint
        state.web.showHspFile = data.config.web.showHspFile
        state.web.showSubtitlesFile = data.config.web.showSubtitlesFile
        state.web.sceneTrailerlist = data.config.web.sceneTrailerlist
        state.web.showScriptHeatmap = data.config.web.showScriptHeatmap
        state.web.showAllHeatmaps = data.config.web.showAllHeatmaps
        state.web.updateCheck = data.config.web.updateCheck
        state.web.isAvailOpacity = data.config.web.isAvailOpacity 
        state.web.showOpenInNewWindow = data.config.web.showOpenInNewWindow
        state.web.sceneCardAspectRatio = data.config.web.sceneCardAspectRatio
        state.web.sceneCardScaleToFit = data.config.web.sceneCardScaleToFit
        state.web.actorCardAspectRatio = data.config.web.actorCardAspectRatio
        state.web.actorCardScaleToFit = data.config.web.actorCardScaleToFit
        state.loading = false
      })
  },
  async save ({ state }) {
    state.loading = true
    ky.put('/api/options/interface/web', { json: { ...state.web } })
      .json()
      .then(data => {
        state.web.tagSort = data.tagSort
        state.web.sceneHidden = data.sceneHidden
        state.web.sceneWatchlist = data.sceneWatchlist
        state.web.sceneFavourite = data.sceneFavourite
        state.web.sceneWishlist = data.sceneWishlist
        state.web.sceneWatched = data.sceneWatched
        state.web.sceneEdit = data.sceneEdit
        state.web.sceneDuration = data.sceneDuration
        state.web.sceneCuepoint = data.sceneCuepoint
        state.web.showHspFile = data.showHspFile
        state.web.showSubtitlesFile = data.showSubtitlesFile
        state.web.sceneTrailerlist = data.sceneTrailerlist
        state.web.showScriptHeatmap = data.showScriptHeatmap
        state.web.showAllHeatmaps = data.showAllHeatmaps
        state.web.updateCheck = data.updateCheck
        state.web.isAvailOpacity = data.isAvailOpacity
        state.web.showOpenInNewWindow = data.showOpenInNewWindow
        state.web.sceneCardAspectRatio = data.sceneCardAspectRatio
        state.web.sceneCardScaleToFit = data.sceneCardScaleToFit
        state.web.actorCardAspectRatio = data.actorCardAspectRatio
        state.web.actorCardScaleToFit = data.actorCardScaleToFit
        state.loading = false
      })
  }
}

export default {
  namespaced: true,
  state,
  mutations,
  actions
}
