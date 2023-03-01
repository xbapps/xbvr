import ky from 'ky'

const state = {
  items: []
}

const mutations = {
}

const actions = {
  async load ({ state }, params) {
    state.items = await ky.get('/api/options/sites').json()
  },
  async toggleSite ({ state }, params) {
    state.items = await ky.put(`/api/options/sites/${params.id}`, { json: {} }).json()
  },
  async toggleSubscribed ({ state }, params) {
    state.items = await ky.put(`/api/options/sites/subsrcibed/${params.id}`, { json: {} }).json()
    console.log('calling',params.id)
  }
}

export default {
  namespaced: true,
  state,
  mutations,
  actions
}
