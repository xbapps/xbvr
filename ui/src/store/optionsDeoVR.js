import ky from 'ky'

const state = {
  loading: false,
  deovr: {
    enabled: false,
    auth_enabled: false,
    render_heatmaps: false,
    remote_enabled: false,
    username: '',
    password: '',
    boundIp: []
  }
}

const mutations = {}

const actions = {
  async load ({ state }, params) {
    state.loading = true
    ky.get('/api/options/state')
      .json()
      .then(data => {
        state.deovr.enabled = data.config.interfaces.deovr.enabled
        state.deovr.auth_enabled = data.config.interfaces.deovr.auth_enabled
        state.deovr.render_heatmaps = data.config.interfaces.deovr.render_heatmaps
        state.deovr.remote_enabled = data.config.interfaces.deovr.remote_enabled
        state.deovr.username = data.config.interfaces.deovr.username
        state.deovr.password = data.config.interfaces.deovr.password
        state.deovr.boundIp = data.currentState.server.bound_ip
        state.loading = false
      })
  },
  async save ({ state }, enabled) {
    state.loading = true
    ky.put('/api/options/interface/deovr', { json: { ...state.deovr } })
      .json()
      .then(data => {
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
