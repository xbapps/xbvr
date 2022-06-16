<template>
  <a :class="buttonClass"
     @click="$store.commit('sceneList/toggleSceneList', {scene_id: item.scene_id, list: 'watched'})"
     :title="item.is_watched ? 'Mark as unwatched' : 'Mark as watched'">
    <b-icon pack="mdi" :icon="item.is_watched ? 'eye-check' : 'eye'" size="is-small"/>
  </a>
</template>

<script>
export default {
  name: 'WatchedButton',
  props: { item: Object },
  computed: {
    buttonClass () {
      if (this.item.is_watched) {
        return 'button is-dark is-info is-small'
      }
      return 'button is-dark is-info is-outlined is-small'
    }
  }
}
</script>
