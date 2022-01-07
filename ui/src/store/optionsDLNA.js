import ky from 'ky'

const state = {
  loading: false,
  dlna: {
    enabled: false,
    name: '',
    image: '',
    allowedIp: [],
    recentIp: [],
    availableImages: []
  }
}

const mutations = {}

const actions = {
  async load ({ state }, params) {
    state.loading = true
    ky.get('/api/options/state')
      .json()
      .then(data => {
        state.dlna.enabled = data.currentState.dlna.running
        state.dlna.availableImages = data.currentState.dlna.images
        state.dlna.recentIp = data.currentState.dlna.recentIp
        state.dlna.name = data.config.interfaces.dlna.serviceName
        state.dlna.image = data.config.interfaces.dlna.serviceImage
        state.dlna.allowedIp = data.config.interfaces.dlna.allowedIp
        state.loading = false
      })
  },
  async save ({ state }, enabled) {
    state.loading = true
    ky.put('/api/options/interface/dlna', { json: { ...state.dlna } })
      .json()
      .then(data => {
        state.dlna.enabled = data.enabled
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
