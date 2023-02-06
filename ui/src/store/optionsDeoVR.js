import ky from 'ky'

const state = {
  loading: false,
  deovr: {
    enabled: false,
    auth_enabled: false,
    render_heatmaps: false,
    track_watch_time: true,
    remote_enabled: false,
    username: '',
    password: '',
    boundIp: []
  },
  heresphere: {
    allow_file_deletes: false,
    allow_rating_updates: false,
    allow_favorite_updates: false,
    allow_tag_updates: false,
    allow_cuepoint_updates: false,
    allow_watchlist_updates: false,
    allow_hsp_data: false,
    multitrack_cuepoints: true,
    multitrack_cast_cuepoints: true,
    retain_non_hsp_cuepoints: true
  },
  players: {
    video_sort_seq: '',
    script_sort_seq: ''
  }
}

const mutations = {}

const actions = {
  async load ({ state }, params) {
    state.loading = true
    ky.get('/api/options/state')
      .json()
      .then(data => {
        state.deovr.enabled = data.config.interfaces.deovr.enabled
        state.deovr.auth_enabled = data.config.interfaces.deovr.auth_enabled
        state.deovr.render_heatmaps = data.config.interfaces.deovr.render_heatmaps
        state.deovr.track_watch_time = data.config.interfaces.deovr.track_watch_time
        state.deovr.remote_enabled = data.config.interfaces.deovr.remote_enabled
        state.deovr.username = data.config.interfaces.deovr.username
        state.deovr.password = data.config.interfaces.deovr.password
        state.deovr.boundIp = data.currentState.server.bound_ip
        state.heresphere.allow_file_deletes = data.config.interfaces.heresphere.allow_file_deletes
        state.heresphere.allow_rating_updates = data.config.interfaces.heresphere.allow_rating_updates
        state.heresphere.allow_favorite_updates = data.config.interfaces.heresphere.allow_favorite_updates
        state.heresphere.allow_tag_updates = data.config.interfaces.heresphere.allow_tag_updates
        state.heresphere.allow_cuepoint_updates = data.config.interfaces.heresphere.allow_cuepoint_updates
        state.heresphere.allow_watchlist_updates = data.config.interfaces.heresphere.allow_watchlist_updates
        state.heresphere.allow_hsp_data = data.config.interfaces.heresphere.allow_hsp_data
        state.heresphere.multitrack_cuepoints = data.config.interfaces.heresphere.multitrack_cuepoints
        state.players.video_sort_seq = data.config.interfaces.players.video_sort_seq
        state.players.script_sort_seq = data.config.interfaces.players.script_sort_seq
        state.heresphere.multitrack_cast_cuepoints = data.config.interfaces.heresphere.multitrack_cast_cuepoints
        state.heresphere.retain_non_hsp_cuepoints = data.config.interfaces.heresphere.retain_non_hsp_cuepoints
        state.loading = false        
      })
  },
  async save ({ state }, enabled) {
    state.loading = true
    ky.put('/api/options/interface/deovr', { json: { ...state.deovr, ...state.heresphere, ...state.players } })
      .json()
      .then(data => {
        state.loading = false
      })
  }
}

export default {
  namespaced: true,
  state,
  mutations,
  actions
}
