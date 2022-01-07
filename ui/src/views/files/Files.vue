<template>
  <div>
    <div class="container is-fluid">
      <div>
        <Filters/>
      </div>
      <div class="columns">
        <div class="column">
          <List/>
        </div>
      </div>
    </div>
    <Player v-if="showPlayerOverlay"/>
    <SceneMatch v-if="showMatchOverlay"/>
  </div>
</template>

<script>
import Filters from './Filters'
import List from './List'
import Player from './Player'
import SceneMatch from './SceneMatch'

export default {
  name: 'Files',
  components: { Filters, List, Player, SceneMatch },
  computed: {
    showPlayerOverlay () {
      return this.$store.state.overlay.player.show
    },
    showMatchOverlay () {
      return this.$store.state.overlay.match.show
    }
  }
}
</script>
