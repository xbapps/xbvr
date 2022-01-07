const state = {
  isPreviewReady: false,
  generatingPreview: false,
  previewFn: ''
}

const mutations = {
  hidePreview (state) {
    state.isPreviewReady = false
    state.generatingPreview = true
    state.previewFn = ''
  },
  showPreview (state, payload) {
    state.isPreviewReady = true
    state.generatingPreview = false
    state.previewFn = payload.previewFn
  }
}

export default {
  namespaced: true,
  state,
  mutations
}
