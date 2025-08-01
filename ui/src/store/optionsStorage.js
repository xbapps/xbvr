import ky from 'ky'

const state = {
  items: [],
  options: {
    match_ohash: false,
    forbidden_video_ext: [],
    video_ext: [],
    default_video_ext: [],
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
      state.options.forbidden_video_ext = data.forbidden_video_ext
      state.options.video_ext = data.video_ext
      state.options.default_video_ext = data.default_video_ext
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
