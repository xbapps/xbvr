import ky from "ky";

const state = {
  items: [],
};

const actions = {
  async load({state}, params) {
    state.items = await ky.get(`/api/config/volume`).json();
  },
};


export default {
  namespaced: true,
  state,
  actions,
}
