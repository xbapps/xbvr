import ky from 'ky'

const state = {
  items: [],
  options: {
    match_ohash: false,
    video_ext: []
  },  
  loading: false,
}

const mutations = {
  setItems(state, payload) {
    state.items = payload;
  },
  setOptions(state, payload) {
    state.options = payload;
  },
  setLoading(state, payload) {
    state.loading = payload;
  },
}

const actions = {
  async load({commit}) {
    commit('setLoading', true);
    const data = await ky.get(`/api/options/storage`).json();
    commit('setItems', data.volumes || []);
    commit('setOptions', {
      match_ohash: data.match_ohash || false,
      video_ext: data.video_ext || []
    });
    commit('setLoading', false);
  },
  async save({state}) {
    await ky.put("/api/options/storage", {
      json: {
        match_ohash: state.options.match_ohash,
        video_ext: state.options.video_ext
      }
    });
  },  
}

export default {
  namespaced: true,
  state,
  mutations,
  actions
}
