import ky from "ky";

const state = {
  loading: false,
  prefs: {
    tagSort: 'By Tag Count',
  },
};

const mutations = {};

const actions = {
  async load({state}) {
    state.loading = true;
    ky.get(`/api/options/state`)
      .json()
      .then(data => {
        state.prefs.tagSort = data.currentState.preferences.tagSort;
        state.loading = false;
      });
  },
  async save({state}) {
    state.loading = true;
    ky.put(`/api/options/preferences`, {json: {...state.prefs}})
      .json()
      .then(data => {
        state.prefs.tagSort = data.tagSort;
        state.loading = false;
      });
  }
};

export default {
  namespaced: true,
  state,
  mutations,
  actions,
}
