<template>
  <a :class="buttonClass"
     @click="$store.commit('sceneList/toggleSceneList', {scene_id: item.scene_id, list: 'watchlist'})"
     :title="item.watchlist ? 'Remove from watchlist' : 'Add to watchlist'">
    <b-icon pack="mdi" :icon="item.watchlist ? 'calendar-check' : 'calendar-blank'" size="is-small"/>
  </a>
</template>

<script>
export default {
  name: 'WatchlistButton',
  props: { item: Object },
  computed: {
    buttonClass () {
      if (this.item.watchlist) {
        return 'button is-primary is-small'
      }
      return 'button is-primary is-outlined is-small'
    }
  }
}
</script>
