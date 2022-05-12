<template>
  <a :class="buttonClass"
     @click="$store.commit('sceneList/toggleSceneList', {scene_id: item.scene_id, list: 'favourite'})"
     :title="item.favourite ? 'Remove from favourites' : 'Add to favourites'">
    <b-icon pack="mdi" :icon="item.favourite ? 'heart' : 'heart-outline'" size="is-small"/>
  </a>
</template>

<script>
export default {
  name: 'FavouriteButton',
  props: { item: Object },
  computed: {
    buttonClass () {
      if (this.item.favourite) {
        return 'button is-danger is-small'
      }
      return 'button is-danger is-outlined is-small'
    }
  }
}
</script>
