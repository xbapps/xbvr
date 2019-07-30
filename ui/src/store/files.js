import ky from "ky";

const state = {
  items: [],
};

const actions = {
  load({state}, params) {
    ky.get(`/api/files/list/unmatched`).json().then(data => {
      state.items = data;
    });
  },
};


export default {
  namespaced: true,
  state,
  actions,
}
