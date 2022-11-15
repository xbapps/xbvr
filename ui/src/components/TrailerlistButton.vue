<template>
  <a :class="buttonClass"
   v-if="!item.is_available & item.trailer_source !=='' & this.$store.state.optionsWeb.web.sceneTrailerlist"
     @click="$store.commit('sceneList/toggleSceneList', {scene_id: item.scene_id, list: 'trailerlist'})"
     :title="item.watchlist ? 'Remove from Trailer List' : 'Add to Trailer List'">
    <b-icon pack="mdi" :icon="item.watchlist ? 'movie-search-outline' : 'movie-search-outline'" size="is-small"/>
  </a>
</template>

<script>
export default {
  name: 'TrailerlistButton',
  props: { item: Object },
  computed: {
    buttonClass () {
      if (this.item.trailerlist) {
        return 'button is-primary is-small'
      }
      return 'button is-primary is-outlined is-small'
    }
  }
}
</script>
