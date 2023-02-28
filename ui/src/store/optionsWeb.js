import ky from 'ky'

const state = {
  loading: false,
  web: {
    tagSort: 'By Tag Count',
    sceneWatchlist: true,
    sceneFavourite: true,
    sceneWatched: false,
    sceneEdit: false,
    sceneCuepoint: true,
    showHspFile: true,
    showSubtitlesFile: true,
    sceneTrailerlist: true,
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
        state.web.sceneWatchlist = data.config.web.sceneWatchlist
        state.web.sceneFavourite = data.config.web.sceneFavourite
        state.web.sceneWatched = data.config.web.sceneWatched
        state.web.sceneEdit = data.config.web.sceneEdit
        state.web.sceneCuepoint = data.config.web.sceneCuepoint
        state.web.showHspFile = data.config.web.showHspFile
        state.web.showSubtitlesFile = data.config.web.showSubtitlesFile
        state.web.sceneTrailerlist = data.config.web.sceneTrailerlist
        state.web.updateCheck = data.config.web.updateCheck
        state.loading = false
      })
  },
  async save ({ state }) {
    state.loading = true
    ky.put('/api/options/interface/web', { json: { ...state.web } })
      .json()
      .then(data => {
        state.web.tagSort = data.tagSort
        state.web.sceneWatchlist = data.sceneWatchlist
        state.web.sceneFavourite = data.sceneFavourite
        state.web.sceneWatched = data.sceneWatched
        state.web.sceneEdit = data.sceneEdit
        state.web.sceneCuepoint = data.sceneCuepoint
        state.web.showHspFile = data.showHspFile
        state.web.showSubtitlesFile = data.showSubtitlesFile        
        state.web.sceneTrailerlist = data.sceneTrailerlist
        state.web.updateCheck = data.updateCheck
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
