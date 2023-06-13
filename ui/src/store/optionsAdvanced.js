import ky from 'ky'

const state = {
  loading: false,
  advanced: {
    showInternalSceneId: false,
    showHSPApiLink: false,
    stashApiKey: '',
    scrapeActorAfterScene: 'true',
    useImperialEntry: 'false',
  }
}

const mutations = {}

const actions = {
  async load ({ state }) {
    state.loading = true
    ky.get('/api/options/state')
      .json()
      .then(data => {
        state.advanced.showInternalSceneId = data.config.advanced.showInternalSceneId
        state.advanced.showHSPApiLink = data.config.advanced.showHSPApiLink
        state.advanced.stashApiKey = data.config.advanced.stashApiKey
        state.advanced.scrapeActorAfterScene = data.config.advanced.scrapeActorAfterScene
        state.advanced.useImperialEntry = data.config.advanced.useImperialEntry
        state.loading = false
      })
  },
  async save ({ state }) {
    state.loading = true
    ky.put('/api/options/interface/advanced', { json: { ...state.advanced } })
      .json()
      .then(data => {
        state.advanced.showInternalSceneId = data.showInternalSceneId
        state.advanced.showHSPApiLink = data.showHSPApiLink
        state.advanced.stashApiKey = data.stashApiKey
        state.advanced.scrapeActorAfterScene = data.scrapeActorAfterScene
        state.advanced.useImperialEntry = data.useImperialEntry
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
