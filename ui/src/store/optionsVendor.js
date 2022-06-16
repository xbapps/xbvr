import ky from 'ky'

const state = {
  tpdb: {
    apiToken: ''
  }
}

const mutations = {}

const actions = {
  async load({ state }, params) {
    ky.get('/api/options/state')
      .json()
      .then(data => {
        state.tpdb.apiToken = data.config.vendor.tpdb.apiToken
      })
  },
}

export default {
  namespaced: true,
  state,
  mutations,
  actions
}
