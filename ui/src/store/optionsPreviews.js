const state = {
  isPreviewReady: false,
  generatingPreview: false,
  previewFn: '',
  previewError: '',
  previewStartTime: null,
  previewElapsed: 0
}

const mutations = {
  hidePreview (state) {
    state.isPreviewReady = false
    state.generatingPreview = true
    state.previewFn = ''
    state.previewError = ''
    state.previewStartTime = Date.now()
    state.previewElapsed = 0
  },
  showPreview (state, payload) {
    state.isPreviewReady = true
    state.generatingPreview = false
    state.previewFn = payload.previewFn
    state.previewError = ''
    state.previewStartTime = null
  },
  previewFailed (state, payload) {
    state.isPreviewReady = false
    state.generatingPreview = false
    state.previewFn = ''
    state.previewError = (payload && payload.message) || 'Preview generation failed'
    state.previewStartTime = null
  },
  tickPreviewTimer (state) {
    if (state.previewStartTime) {
      state.previewElapsed = Math.floor((Date.now() - state.previewStartTime) / 1000)
    }
  },
  clearPreview (state) {
    state.isPreviewReady = false
    state.generatingPreview = false
    state.previewFn = ''
    state.previewError = ''
    state.previewStartTime = null
    state.previewElapsed = 0
  }
}

export default {
  namespaced: true,
  state,
  mutations
}
