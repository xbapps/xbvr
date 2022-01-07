import ky from 'ky'

const state = {
  items: []
}

const mutations = {
}

const actions = {
  async load ({ state }, params) {
    state.items = await ky.get('/api/options/storage').json()
  }
}

export default {
  namespaced: true,
  state,
  mutations,
  actions
}
