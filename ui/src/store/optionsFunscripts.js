import ky from 'ky'

const state = {
  countTotal: 0,
  countUpdated: 0,  
  optionsFunscripts: {
    scrapeFunscripts: false,
  }
}

const mutations = {}

const actions = {
  async load({ state }, params) {
    ky.get('/api/options/funscripts/count')
      .json()
      .then(data => {
        state.countTotal = data.total
        state.countUpdated = data.updated
      })
    ky.get('/api/options/state')
      .json()
      .then(data => {
        state.optionsFunscripts.scrapeFunscripts = data.config.funscripts.scrapeFunscripts
      })

  },
  async save ({ state }) {    
    ky.put('/api/options/funscripts', { json: { ...state.optionsFunscripts } })
      .json()
      .then(data => {
        state.optionsFunscripts.scrapeFunscripts = data.scrapeFunscripts
      })
  },
}

export default {
  namespaced: true,
  state,
  mutations,
  actions
}
