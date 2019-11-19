import ky from "ky";

const state = {
  items: [],
};

const actions = {
  async load({state}, params) {
    state.items = await ky.get(`/api/config/storage`).json();
  },
};


export default {
  namespaced: true,
  state,
  actions,
}
