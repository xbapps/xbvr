<template>
  <a :class="buttonClass"
    @click="$store.commit('sceneList/toggleSceneList', {scene_id: item.scene_id, list: 'needs_update'})"
     :title="item.needs_update ? 'Do not refresh scene' : 'Refresh scene on next scrape'">
    <b-icon pack="mdi" icon="refresh" size="is-small"/>
  </a>
</template>

<script>
export default {
  name: 'RefreshButton',
  props: { item: Object },
  computed: {
    buttonClass () {
      if (this.item.needs_update) {
        return 'button is-dark is-small'
      }
      return 'button is-dark is-outlined is-small'
    }
  }
}
</script>