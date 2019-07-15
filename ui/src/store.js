import Vue from "vue";
import Vuex from "vuex";
import ky from "ky";

Vue.use(Vuex);

export default new Vuex.Store({
  state: {
    messages: {
      lockScrape: false,
      lastScrapeMessage: "",
      lockRescan: false,
      lastRescanMessage: "",
    },
    sceneList: {
      items: [],
      offset: 0,
      total: 0,
      limit: 80,
      filterOpts: {},
      filters: {
        dlState: "available",
        isAvailable: "1",
        isAccessible: "1",
        cardSize: 1,
        releaseMonth: "",
        cast: "",
        site: "",
        tag: "",
      }
    },
    detailsOverlay: {
      show: false,
      scene: null,
    }
  },
  mutations: {
    toggleSceneList(state, payload) {
      state.sceneList.items.map(obj => {
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
    showDetailsOverlay(state, payload) {
      state.detailsOverlay.scene = payload.scene;
      state.detailsOverlay.show = true;
    },
    hideDetailsOverlay(state, payload) {
      state.detailsOverlay.scene = null;
      state.detailsOverlay.show = false;
    }
  },
  actions: {
    async loadFilters({state}) {
      state.sceneList.filterOpts = await ky
        .get(`/api/scene/filters/state`, {
          searchParams: {
            is_available: state.sceneList.filters.isAvailable,
            is_accessible: state.sceneList.filters.isAccessible,
          }
        }).json();

      // Reverse list of release months for display purposes
      state.sceneList.filterOpts.release_month = state.sceneList.filterOpts.release_month.reverse()
    },
    async loadList({state}, params) {
      let iOffset = params.offset || 0;

      let data = await ky
        .get(`/api/scene/list`, {
          searchParams: {
            offset: iOffset,
            limit: state.sceneList.limit,
            is_available: state.sceneList.filters.isAvailable,
            is_accessible: state.sceneList.filters.isAccessible,
            tag: state.sceneList.filters.tag,
            cast: state.sceneList.filters.cast,
            site: state.sceneList.filters.site,
            released: state.sceneList.filters.releaseMonth,
          }
        })
        .json();

      if (iOffset === 0) {
        state.sceneList.items = [];
      }

      state.sceneList.items = state.sceneList.items.concat(data.scenes);
      state.sceneList.offset = iOffset + state.sceneList.limit;
      state.sceneList.total = data.results;
    },
  },
})
