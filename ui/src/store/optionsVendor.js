import ky from 'ky'

const state = {
  tpdb: {
    apiToken: '',
  },
  scrapers: [],
}

const mutations = {}

const actions = {
  async load({ state }, params) {
    ky.get('/api/options/state')
      .json()
      .then(data => {
        state.tpdb.apiToken = data.config.vendor.tpdb.apiToken
        state.scrapers = data.scrapers        
      })
  },
}

export default {
  namespaced: true,
  state,
  mutations,
  actions
}
