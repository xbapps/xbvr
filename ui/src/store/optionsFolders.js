import ky from "ky";

const state = {
  items: [],
};

const actions = {
  async load({state}, params) {
    state.items = await ky.get(`/api/config/folder`).json();
  },
};


export default {
  namespaced: true,
  state,
  actions,
}
