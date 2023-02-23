<template>
  <a :class="buttonClass"
     @click="toggleState()"
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
  },
  methods: {
    toggleState() {
      let currentToggle=this.item.favourite
      this.$store.commit('sceneList/toggleSceneList', {scene_id: this.item.scene_id, list: 'favourite'})
      this.item.favourite=!currentToggle
    }
  }
}
</script>
