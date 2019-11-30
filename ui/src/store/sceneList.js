import ky from "ky";
import Vue from "vue";

function defaultValue(v, d) {
  if (v === undefined) {
    return d;
  }
  return v;
}

const defaultFilterState = {
  dlState: "available",
  cardSize: "1",

  lists: [],
  isAvailable: true,
  isAccessible: true,
  isWatched: null,
  releaseMonth: "",
  cast: [],
  sites: [],
  tags: [],
  cuepoint: [],
  sort: "release_desc",
};

const state = {
  items: [],
  isLoading: false,
  offset: 0,
  total: 0,
  limit: 80,
  filterOpts: {
    cast: [],
    sites: [],
    tags: [],
  },
  filters: defaultFilterState
};

const getters = {
  filterQueryParams: (state) => {
    const st = Object.assign({}, state.filters);
    delete st.cardSize;

    return Buffer.from(JSON.stringify(st)).toString("base64");
  },
  prevScene: (state) => (currentScene) => {
    let i = state.items.findIndex(item => item.scene_id == currentScene.scene_id);
    if (i === 0) {
      return null;
    }
    return state.items[i - 1];
  },
  nextScene: (state) => (currentScene) => {
    let i = state.items.findIndex(item => item.scene_id == currentScene.scene_id);
    if (i === state.items.length - 1) {
      return null;
    }
    return state.items[i + 1];
  },
};

const mutations = {
  toggleSceneList(state, payload) {
    state.items = state.items.map(obj => {
      if (obj.scene_id === payload.scene_id) {
        if (payload.list === "watchlist") {
          obj.watchlist = !obj.watchlist;
        }
        if (payload.list === "favourite") {
          obj.favourite = !obj.favourite;
        }
      }
      return obj;
    });

    ky.post(`/api/scene/toggle`, {
      json: {
        scene_id: payload.scene_id,
        list: payload.list,
      }
    });
  },
  updateScene(state, payload) {
    state.items = state.items.map(obj => {
      if (obj.scene_id === payload.scene_id) {
        obj = payload;
      }
      return obj;
    })
  },
  stateFromQuery(state, payload) {
    try {
      const obj = JSON.parse(Buffer.from(payload.q, "base64").toString("utf-8"));
      for (let [k, v] of Object.entries(obj)) {
        Vue.set(state.filters, k, v)
      }
    } catch (err) {
    }
  }
};

const actions = {
  async filters({state}) {
    state.filterOpts = await ky.get(`/api/scene/filters`).json();

    // Reverse list of release months for display purposes
    state.filterOpts.release_month = state.filterOpts.release_month.reverse()
  },
  async load({state, getters}, params) {
    let iOffset = params.offset || 0;

    state.isLoading = true;

    let q = Object.assign({}, state.filters);
    q.offset = iOffset;
    q.limit = state.limit;

    let data = await ky
      .post(`/api/scene/list`, {json: q})
      .json();

    state.isLoading = false;

    if (iOffset === 0) {
      state.items = [];
    }

    state.items = state.items.concat(data.scenes);
    state.offset = iOffset + state.limit;
    state.total = data.results;
  },
};

export default {
  namespaced: true,
  state,
  getters,
  mutations,
  actions,
}
