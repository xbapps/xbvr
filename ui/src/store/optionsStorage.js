import ky from 'ky'

const state = {
  items: [],
  options: {
    match_ohash: false,
  },  
}

const mutations = {
}

const actions = {
  async load ({ state }, params) {
    await ky.get('/api/options/storage').json()
    .then(data => {
      state.items = data.volumes
      state.options.match_ohash = data.match_ohash
    })
  },
  async save ({ state }, enabled) { 
    ky.put('/api/options/storage', { json: { ...state.options } })      
  },  
}

export default {
  namespaced: true,
  state,
  mutations,
  actions
}
