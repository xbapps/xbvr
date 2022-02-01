import ky from 'ky'

const state = {
  loading: false,
  web: {
    tagSort: 'By Tag Count',
    sceneEdit: false,
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
        state.web.sceneEdit = data.config.web.sceneEdit
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
        state.web.sceneEdit = data.sceneEdit
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
