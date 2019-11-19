import ky from "ky";
import Vue from "vue";

function defaultValue(v, d) {
  if (v === undefined) {
    return d;
  }
  return v;
}

const defaultFilterState = {
  cardSize: "1",
}

const state = {
  items: [],
  isLoading: false,
  offset: 0,
  total: 0,
  limit: 80,
  filters: defaultFilterState
};

const actions = {
  async load({ state, getters }, params) {
    let iOffset = params.offset || 0;

    state.isLoading = true;

    let q = Object.assign({}, state.filters);
    q.offset = iOffset;
    q.limit = state.limit;

    let data = await ky
      .post(`/api/actor/list`, { json: q })
      .json();

    state.isLoading = false;

    if (iOffset === 0) {
      state.items = [];
    }

    state.items = state.items.concat(data.actors);
    state.offset = iOffset + state.limit;
    state.total = data.results;
  },
};

export default {
  namespaced: true,
  state,
  actions,
}
