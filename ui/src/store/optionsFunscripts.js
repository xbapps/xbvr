import ky from 'ky'

const state = {
  count: 0,
}

const mutations = {}

const actions = {
  async load ({ state }, params) {
    state.count = await ky.get('/api/options/funscripts/count').json()
  },
}

export default {
  namespaced: true,
  state,
  mutations,
  actions
}
