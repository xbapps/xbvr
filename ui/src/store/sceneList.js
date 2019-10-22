import ky from "ky";

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
  filters: {
    dlState: "available",
    isAvailable: "1",
    isAccessible: "1",
    isWatched: "",
    cardSize: "1",
    releaseMonth: "",
    cast: [],
    sites: [],
    tags: [],
    sort: "release_desc",
  }
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
  async load({state}, params) {
    let iOffset = params.offset || 0;

    state.isLoading = true;

    let data = await ky
      .get(`/api/scene/list`, {
        searchParams: {
          offset: iOffset,
          limit: state.limit,
          is_available: state.filters.isAvailable,
          is_accessible: state.filters.isAccessible,
          is_watched: state.filters.isWatched,
          tags: state.filters.tags.join(","),
          cast: state.filters.cast.join(","),
          sites: state.filters.sites.join(","),
          released: state.filters.releaseMonth,
          sort: state.filters.sort,
        }
      })
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
  mutations,
  actions,
}
