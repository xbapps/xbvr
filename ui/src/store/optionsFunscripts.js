import ky from 'ky'

const state = {
  countTotal: 0,
  countUpdated: 0
}

const mutations = {}

const actions = {
  async load({ state }, params) {
    ky.get('/api/options/funscripts/count')
      .json()
      .then(data => {
        state.countTotal = data.total
        state.countUpdated = data.updated
      })
  },
}

export default {
  namespaced: true,
  state,
  mutations,
  actions
}
