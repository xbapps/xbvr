import ky from "ky";

function defaultValue(v, d) {
  if (v === undefined) {
    return d;
  }
  return v;
}

const defaultFilterState = {
  dlState: "available",
  lists: [],
  isAvailable: "1",
  isAccessible: "1",
  isWatched: "",
  cardSize: "1",
  releaseMonth: "",
  cast: [],
  sites: [],
  tags: [],
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
    return {
      lists: state.filters.lists.join(","),
      is_available: state.filters.isAvailable,
      is_accessible: state.filters.isAccessible,
      is_watched: state.filters.isWatched,
      tags: state.filters.tags.join(","),
      cast: state.filters.cast.join(","),
      sites: state.filters.sites.join(","),
      released: state.filters.releaseMonth,
      sort: state.filters.sort,
    }
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
      state.filters.lists = defaultValue(payload.lists.split(",").filter(el => el !== ""), defaultFilterState.lists);
      state.filters.isAvailable = defaultValue(payload.is_available, defaultFilterState.isAvailable);
      state.filters.isAccessible = defaultValue(payload.is_accessible, defaultFilterState.isAccessible);
      state.filters.isWatched = defaultValue(payload.is_watched, defaultFilterState.isWatched);
      state.filters.tags = defaultValue(payload.tags.split(",").filter(el => el !== ""), defaultFilterState.tags);
      state.filters.cast = defaultValue(payload.cast.split(",").filter(el => el !== ""), defaultFilterState.cast);
      state.filters.sites = defaultValue(payload.sites.split(",").filter(el => el !== ""), defaultFilterState.sites);
      state.filters.releaseMonth = defaultValue(payload.released, defaultFilterState.releaseMonth);
      state.filters.sort = defaultValue(payload.sort, defaultFilterState.sort);
    } catch (err) {
      state.filters = defaultFilterState;
    }
  }
};

const actions = {
  async filters({state}) {
    state.filterOpts = await ky
      .get(`/api/scene/filters/state`, {
        searchParams: {
          is_available: state.filters.isAvailable,
          is_accessible: state.filters.isAccessible,
        }
      }).json();

    // Reverse list of release months for display purposes
    state.filterOpts.release_month = state.filterOpts.release_month.reverse()
  },
  async load({state, getters}, params) {
    let iOffset = params.offset || 0;

    state.isLoading = true;

    let q = Object.assign({}, getters.filterQueryParams);
    q.offset = iOffset;
    q.limit = state.limit;

    let data = await ky
      .get(`/api/scene/list`, {searchParams: q})
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
