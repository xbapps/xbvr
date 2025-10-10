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

    <Details v-if="showOverlay"/>
    <EditScene v-if="showEdit" />
    <ActorDetails v-if="showActorDetails"/>
    <EditActor v-if="showActorEdit" />
    <SearchStashdbScenes v-if="showSearchStashdbScenes" />
    <SearchStashdbActors v-if="showSearchStashdbActors" />

    <QuickFind/>
    <MigrationOverlay/>

    <Socket/>
  </div>
</template>

<script>
import GlobalEvents from 'vue-global-events'

import Navbar from './Navbar.vue'
import Socket from './Socket.vue'
import QuickFind from './QuickFind'
import Details from './views/scenes/Details'
import EditScene from './views/scenes/EditScene'
import ActorDetails from './views/actors/ActorDetails'
import EditActor from './views/actors/EditActor'
import SearchStashdbScenes from './views/scenes/SearchStashdbScenes'
import SearchStashdbActors from './views/actors/SearchStashdbActors'
import MigrationOverlay from './components/MigrationOverlay'

export default {
  components: { Navbar, Socket, QuickFind, GlobalEvents, Details, EditScene, ActorDetails, EditActor, SearchStashdbScenes, SearchStashdbActors, MigrationOverlay },
  computed: {
    showOverlay () {
      return this.$store.state.overlay.details.show
    },
    showEdit () {
      return this.$store.state.overlay.edit.show
    },
    showActorDetails() {
      return this.$store.state.overlay.actordetails.show
    },
    showActorEdit() {
      return this.$store.state.overlay.actoredit.show
    },
    showSearchStashdbScenes() {
      return this.$store.state.overlay.searchStashDbScenes.show
    },
    showSearchStashdbActors() {
      return this.$store.state.overlay.searchStashDbActors.show
    },
  }
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
