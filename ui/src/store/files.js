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
    if (state.items.length <= 1) {
      return null
    }

    const currentIndex = state.items.findIndex(item => item.id === currentFile.id)
    if (currentIndex === 0) {
      return state.items[state.items.length - 1]
    }
    return state.items[currentIndex - 1]
  },
  nextFile: (state) => (currentFile) => {
    if (state.items.length <= 1) {
      return null
    }

    const currentIndex = state.items.findIndex(item => item.id === currentFile.id)
    if (currentIndex === state.items.length - 1) {
      return state.items[0]
    }
    return state.items[currentIndex + 1]
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
