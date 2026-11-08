import ky from 'ky'

const state = {
  loading: false,
  running: false,
  showIgnored: false,
  done: 0,
  total: 0,
  groups: []
}

const mutations = {
  set (state, data) {
    state.running = data.running
    state.done = data.done || 0
    state.total = data.total || 0
    state.groups = data.groups || []
  },
  setLoading (state, v) { state.loading = v },
  setShowIgnored (state, v) { state.showIgnored = v }
}

const actions = {
  async load ({ state, commit }) {
    commit('setLoading', true)
    try {
      const data = await ky.get('/api/organize/duplicates' + (state.showIgnored ? '?showIgnored=true' : '')).json()
      commit('set', data)
    } finally {
      commit('setLoading', false)
    }
  },
  async analyze (_ctx, force) {
    await ky.post('/api/organize/duplicates/analyze' + (force ? '?force=true' : '')).json()
  },
  async ignore (_ctx, fileId) {
    await ky.post('/api/organize/duplicates/ignore', { json: { fileId } }).json()
  },
  async unignore (_ctx, fileId) {
    await ky.post('/api/organize/duplicates/unignore', { json: { fileId } }).json()
  },
  async del (_ctx, fileIds) {
    await ky.post('/api/organize/duplicates/delete', { json: { fileIds } }).json()
  },
  async disassociate (_ctx, fileIds) {
    await ky.post('/api/organize/duplicates/disassociate', { json: { fileIds } }).json()
  }
}

export default { namespaced: true, state, mutations, actions }
