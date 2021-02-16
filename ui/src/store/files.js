import ky from 'ky'

const state = {
  isLoading: false,
  items: [],
  filters: {
    sort: '',
    state: 'unmatched',
    createdDate: [],
    resolutions: [],
    framerates: [],
    bitrates: [],
    filename: ''
  }
}

const getters = {
  prevFile: (state) => (currentFile) => {
    const i = state.items.findIndex(item => item.id === currentFile.id)
    if (i === 0) {
      return state.items[state.items.length - 1]
    }
    return state.items[i - 1]
  },
  nextFile: (state) => (currentFile) => {
    const i = state.items.findIndex(item => item.id === currentFile.id)
    if (i === state.items.length - 1) {
      return state.items[0]
    }
    return state.items[i + 1]
  }
}

const actions = {
  load ({ state }, params) {
    state.isLoading = true
    ky.post('/api/files/list', { json: state.filters })
      .json()
      .then(data => {
        state.items = data
        state.isLoading = false
      })
  }
}

export default {
  namespaced: true,
  state,
  getters,
  actions
}
