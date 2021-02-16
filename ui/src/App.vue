<template>
  <div>
    <GlobalEvents
      :filter="e => !['INPUT', 'TEXTAREA'].includes(e.target.tagName)"
      @keypress.prevent.questionMark="$store.commit('overlay/showQuickFind')"
    />
    <Navbar/>
    <div class="navbar-pad">
      <router-view/>
    </div>
    <Socket/>
    <QuickFind/>
  </div>
</template>

<script>
import Navbar from './Navbar.vue'
import Socket from './Socket.vue'
import QuickFind from './QuickFind'
import GlobalEvents from 'vue-global-events'

export default {
  components: { Navbar, Socket, QuickFind, GlobalEvents }
}
</script>

<style>
  .navbar-pad {
    margin-top: 1em;
  }
  .modal-background {
    background-color: rgba(0, 0, 0, .40) !important;
  }
</style>
