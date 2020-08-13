import ky from "ky";

const state = {
  loading: false,
  web: {
    tagSort: '',
  },
};

const mutations = {};

const actions = {
  async load({state}) {
    // Doesn't need to be loaded more than once
    if (state.web.tagSort !== '') return;
    state.loading = true;
    ky.get(`/api/options/state`)
      .json()
      .then(data => {
        state.web.tagSort = data.currentState.web.tagSort;
        state.loading = false;
      });
  },
  async save({state}) {
    state.loading = true;
    ky.put(`/api/options/interface/web`, {json: {...state.web}})
      .json()
      .then(data => {
        state.web.tagSort = data.tagSort;
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
