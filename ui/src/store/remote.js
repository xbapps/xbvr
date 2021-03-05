import ky from 'ky'

const state = {
  connected: false,
  deovrHost: '',
  isPlaying: false,
  currentPosition: 0.0,
  sessionStart: '',
  sessionEnd: '',
  currentFileID: 0,
  currentSceneID: 0,

  currentScene: {},
  history: []
}

const mutations = {
  setState (state, payload) {
    const p = ['connected', 'deovrHost', 'isPlaying', 'sessionStart', 'sessionEnd', 'currentFileID', 'currentSceneID', 'currentPosition']
    p.forEach(x => {
      if (payload[x]) {
        state[x] = payload[x]
      }
    })
  },
  setCurrentScene (state, payload) {
    state.currentScene = payload
  },
  addToHistory (state, payload) {
    state.history.push(payload)
  }
}

const actions = {
  async processMessage ({ state, commit }, payload) {
    if (payload.currentSceneID && payload.currentSceneID !== state.currentSceneID) {
      if (state.currentSceneID !== 0) {
        commit('addToHistory', state.currentScene)
      }
      const sceneData = await ky.get(`/api/scene/${payload.currentSceneID}`).json()
      commit('setCurrentScene', sceneData)
    }

    commit('setState', payload)
  }
}

export default {
  namespaced: true,
  actions,
  mutations,
  state
}
