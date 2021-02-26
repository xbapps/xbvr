import Vue from 'vue'
import Router from 'vue-router'

import Options from './views/options/Options'
import Scenes from './views/scenes/Scenes'
import Files from './views/files/Files'

Vue.use(Router)

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
      path: '/files',
      name: 'files',
      component: Files
    },
    {
      path: '/options',
      name: 'options',
      component: Options
    }
  ]
})
