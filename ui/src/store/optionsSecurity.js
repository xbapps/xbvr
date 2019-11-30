import ky from "ky";

const state = {
  dlna_options: {},
};

const actions = {
  async load({ state }, params) {
    ky.get(`/api/security`)
      .json()
      .then(data => {
        state.dlna_options = data.dlna_options;
      });
  },
  async enableDLNA({ state }, enabled) {
    ky.put(`/api/security/enableDLNA`, { json: { "enabled": enabled } })
      .json()
      .then(data => {
        state.dlna_options.dlna_enabled = data.enabled;
      });
  }
};

export default {
  namespaced: true,
  state,
  actions,
}
