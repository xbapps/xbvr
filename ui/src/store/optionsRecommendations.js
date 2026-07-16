import ky from 'ky'

const state = {
  loading: false,
  config: {
    enabled: true,
    useLearnedModel: false,
    modelType: 'linear',
    useVisualEmbeddings: false,
    watchListSize: 30,
    deleteListSize: 30,
    protectRating: 4,
    graceDays: 30,
    excludeRecentlyWatched: true,
    diversityDecay: 0.5,
    wActor: 1.0,
    wTag: 0.7,
    wSite: 0.3,
    wQuality: 0.2,
    wFreshness: 0.2,
    wSize: 0.5,
    wVisualQuality: 0.5,
    vqMaxSamples: 6,
    noiseWeight: 0.3
  }
}

const mutations = {
  setConfig (state, cfg) {
    state.config = { ...state.config, ...cfg }
  },
  setField (state, { key, value }) {
    state.config[key] = value
  },
  setLoading (state, v) {
    state.loading = v
  }
}

const actions = {
  async load ({ commit }) {
    commit('setLoading', true)
    try {
      const data = await ky.get('/api/recommendations/config').json()
      commit('setConfig', data)
    } finally {
      commit('setLoading', false)
    }
  },
  async save ({ state, commit }) {
    commit('setLoading', true)
    try {
      const data = await ky.post('/api/recommendations/config', { json: { ...state.config } }).json()
      commit('setConfig', data)
    } finally {
      commit('setLoading', false)
    }
  },
  async recompute ({ commit }) {
    commit('setLoading', true)
    try {
      await ky.post('/api/recommendations/recompute', { json: {} }).json()
    } finally {
      commit('setLoading', false)
    }
  }
}

export default {
  namespaced: true,
  state,
  mutations,
  actions
}
