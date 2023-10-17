<template>
  <a :class="buttonClass"
     v-if="!item.is_available"
     @click="toggleState()"
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
  },
  methods: {
    toggleState() {
      let currentToggle = this.item.wishlist
      this.$store.commit('sceneList/toggleSceneList', {scene_id: this.item.scene_id, list: 'wishlist'})
      this.item.wishlist = !currentToggle
    }
  }
}
</script>
