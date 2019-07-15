import Vue from 'vue'
import Router from 'vue-router'

import Options from './views/options/Options.vue'
import Scenes from "./views/scenes/Scenes";

Vue.use(Router);


export default new Router({
  mode: 'hash',
  base: process.env.BASE_URL,
  routes: [
    {
      path: '/',
      name: 'scenes',
      component: Scenes
    },
    {
      path: '/options',
      name: 'options',
      component: Options
    }
  ]
})
