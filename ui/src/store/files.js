import ky from "ky";

const state = {
  items: [],
};

const actions = {
  load({state}, params) {
    state.items = ky.get(`/api/files/list/unmatched`).json();
  },
};


export default {
  namespaced: true,
  state,
  actions,
}
