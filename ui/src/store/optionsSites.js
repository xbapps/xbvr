import ky from "ky";

const state = {
  items: [],
};

const actions = {
  async load({state}, params) {
    state.items = await ky.get(`/api/config/sites`).json();
  },
  async toggleSite({state}, params) {
    state.items = await ky.put(`/api/config/sites/${params.id}`, {json: {}}).json();
  }
};


export default {
  namespaced: true,
  state,
  actions,
}
