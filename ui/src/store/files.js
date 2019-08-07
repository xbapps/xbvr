import ky from "ky";

const state = {
  items: [],
  isLoading: false,
};

const actions = {
  load({state}, params) {
    state.isLoading = true;
    ky.get(`/api/files/list/unmatched`).json().then(data => {
      state.items = data;
      state.isLoading = false;
    });
  },
};


export default {
  namespaced: true,
  state,
  actions,
}
