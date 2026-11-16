import ky from 'ky'

const state = {
  loading: false,
  config: {
    baseUrl: '',
    model: '',
    apiKey: '',
    disableThinking: true,
    structuredOutput: 'json_schema',
    timeoutSeconds: 180,
    maxPageChars: 12000,
    braveApiKey: '',
    resultsPerFile: 5,
    minConfidence: 0.5,
    concurrency: 2,
    maxFollowRounds: 2,
    skipDownloadSites: true,
    blockedDomains: '',
    allowPrivateNetworks: false
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
      commit('setConfig', await ky.get('/api/llmscrape/config').json())
    } finally {
      commit('setLoading', false)
    }
  },
  async save ({ state, commit }) {
    commit('setLoading', true)
    try {
      commit('setConfig', await ky.post('/api/llmscrape/config', { json: state.config }).json())
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
