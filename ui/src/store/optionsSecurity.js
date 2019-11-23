import ky from "ky";

const state = {
  dlnaOptions: {
    dlnaEnabled: true,
  },
};

const actions = {
  async enableDLNA({ state }, enabled) {
    state.dlnaOptions.dlnaEnabled = await ky.put(`/api/security/enableDLNA`, { json: { "enabled": enabled } }).json();
  }
};

export default {
  namespaced: true,
  state,
  actions,
}
