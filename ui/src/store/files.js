import ky from "ky";

const state = {
  items: [],
  isLoading: false,
};

const getters = {
  prevFile: (state) => (currentScene) => {
    let i = state.items.findIndex(item => item.id == currentScene.id);
    if (i === 0) {
      return null;
    }
    return state.items[i - 1];
  },
  nextFile: (state) => (currentScene) => {
    let i = state.items.findIndex(item => item.id == currentScene.id);
    if (i === state.items.length - 1) {
      return null;
    }
    return state.items[i + 1];
  },
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
  getters,
  actions,
}
