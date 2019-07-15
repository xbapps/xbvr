import Vue from 'vue';
import App from './App.vue';
import router from './router';
import store from './store';

import Buefy from 'buefy';
import 'buefy/dist/buefy.css';

import 'video.js/dist/video-js.css';
import 'videojs-vr/dist/videojs-vr.css';
// import '@fortawesome/fontawesome-free/css/all.css';
import '@fortawesome/fontawesome-free/js/all';

Vue.config.productionTip = false;
Vue.use(Buefy);

new Vue({
  router,
  store,
  render: h => h(App)
}).$mount('#app');
