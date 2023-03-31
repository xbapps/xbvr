const state = {
  details: {
    show: false,
    scene: null
  },
  edit: {
    show: false,
    scene: null
  },
  player: {
    show: false,
    file: null
  },
  match: {
    show: false,
    file: null
  },
  createScene: {
    show: false,
    file: null
  },
  showQuickFind: false
}

const mutations = {
  showDetails (state, payload) {
    state.details.scene = payload.scene
    state.details.show = true
  },
  hideDetails (state, payload) {
    state.details.scene = null
    state.details.show = false
  },
  editDetails (state, payload) {
    state.edit.scene = payload.scene
    state.edit.show = true
  },
  hideEditDetails (state) {
    state.edit.scene = null
    state.edit.show = false
  },
  showPlayer (state, payload) {
    state.player.file = payload.file
    state.player.show = true
  },
  hidePlayer (state, payload) {
    state.player.file = null
    state.player.show = false
  },
  showMatch (state, payload) {
    state.match.file = payload.file
    state.match.show = true
  },
  hideMatch (state, payload) {
    state.match.file = null
    state.match.show = false
  },
  createCustomScene (state, payload) {
    state.createScene.file = payload.file
    state.createScene.show = true
  },
  hideCreateCustomScene (state, payload) {
    state.createScene.file = null
    state.createScene.show = false    
  },
  showQuickFind (state, payload) {
    state.showQuickFind = true
  },
  hideQuickFind (state, payload) {
    state.showQuickFind = false
  }
}

export default {
  namespaced: true,
  state,
  mutations
}
