import Vue from "vue";
import Vuex from "vuex";

import sceneList from "./sceneList";
import messages from "./messages";
import overlay from "./overlay";


Vue.use(Vuex);

export default new Vuex.Store({
  modules: {
    sceneList,
    messages,
    overlay,
  }
})
