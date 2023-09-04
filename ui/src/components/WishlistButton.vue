<template>
  <a :class="buttonClass"
     v-if="!item.is_available"
     @click="$store.commit('sceneList/toggleSceneList', {scene_id: item.scene_id, list: 'wishlist'})"
     :title="item.wishlist ? 'Remove from wishlist' : 'Add to wishlist'">
    <b-icon pack="mdi" icon="oil-lamp" size="is-small"/>
  </a>
</template>

<script>
export default {
  name: 'WishlistButton',
  props: { item: Object },
  computed: {
    buttonClass () {
      if (this.item.wishlist) {
        return 'button is-info is-small'
      }
      return 'button is-info is-outlined is-small'
    }
  }
}
</script>
