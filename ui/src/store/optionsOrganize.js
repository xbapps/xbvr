import ky from 'ky'

const state = {
  loading: false,
  running: false,
  result: null,
  config: {
    dedup: true,
    deferDups: false,
    incomingDir: 'Incoming',
    incomingMinAge: 30,
    topFolder: '',
    castGender: 'female',
    symlinkByActor: false,
    actorFolder: 'ByActor'
  }
}

const mutations = {
  setConfig (state, cfg) { state.config = { ...state.config, ...cfg } },
  setField (state, { key, value }) { state.config[key] = value },
  setLoading (state, v) { state.loading = v },
  setStatus (state, { running, result }) { state.running = running; if (result) state.result = result }
}

const actions = {
  async load ({ commit }) {
    commit('setLoading', true)
    try {
      const cfg = await ky.get('/api/organize/config').json()
      commit('setConfig', cfg)
      const st = await ky.get('/api/organize/status').json()
      commit('setStatus', st)
    } finally {
      commit('setLoading', false)
    }
  },
  async save ({ state, commit }) {
    commit('setLoading', true)
    try {
      const cfg = await ky.post('/api/organize/config', { json: { ...state.config } }).json()
      commit('setConfig', cfg)
    } finally {
      commit('setLoading', false)
    }
  },
  async run ({ commit }, { dryRun, limit }) {
    await ky.post('/api/organize/run', { json: { dryRun, limit: limit || 0 } }).json()
    commit('setStatus', { running: true })
  },
  async pollStatus ({ commit }) {
    const st = await ky.get('/api/organize/status').json()
    commit('setStatus', st)
    return st
  }
}

export default { namespaced: true, state, mutations, actions }
