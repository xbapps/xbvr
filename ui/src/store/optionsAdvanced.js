import ky from 'ky'

const state = {
  loading: false,
  advanced: {
    showInternalSceneId: false,
    showHSPApiLink: false
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
