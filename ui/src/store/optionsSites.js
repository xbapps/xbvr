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
    state.items = await ky.put(`/api/options/sites/subscribed/${params.id}`, { json: {} }).json()
  },
  async toggleLimitScraping ({ state }, params) {
    state.items = await ky.put(`/api/options/sites/limit_scraping/${params.id}`, { json: {} }).json()
  },
}

export default {
  namespaced: true,
  state,
  mutations,
  actions
}
