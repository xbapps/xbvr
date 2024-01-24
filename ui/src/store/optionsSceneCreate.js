import ky from 'ky'

const state = {
  scrapeScene: '',
  showSceneCreate: false,
}

const mutations = {
  setScrapeScene(state, payload) {
    state.scrapeScene = payload
  },
  showSceneCreate(state, payload) {
    state.showSceneCreate = payload    
  },
}

const actions = {
}

export default {
  namespaced: true,
  state,
  mutations,
  actions
}
