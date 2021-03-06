import Vue from 'vue'
import Vuex from 'vuex'

import sceneList from './sceneList'
import messages from './messages'
import overlay from './overlay'
import files from './files'
import remote from './remote'
import optionsStorage from './optionsStorage'
import optionsWeb from './optionsWeb'
import optionsDLNA from './optionsDLNA'
import optionsDeoVR from './optionsDeoVR'
import optionsSites from './optionsSites'
import optionsPreviews from './optionsPreviews'

Vue.use(Vuex)

export default new Vuex.Store({
  modules: {
    sceneList,
    messages,
    overlay,
    files,
    remote,
    optionsStorage,
    optionsDLNA,
    optionsDeoVR,
    optionsWeb,
    optionsSites,
    optionsPreviews
  }
})
