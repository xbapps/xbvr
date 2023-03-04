<template>
  <a :class="buttonClass"
     @click="toggleState()"
     :title="item.is_hidden ? 'Unhide' : 'Mark as hidden'">
    <b-icon pack="mdi" :icon="item.is_hidden ? 'eye-off-outline' : 'eye-off-outline'" size="is-small"/>
  </a>
</template>

<script>
export default {
  name: 'HiddenButton',
  props: { item: Object },
  computed: {
    buttonClass () {
      if (this.item.is_hidden) {
        return 'button is-danger is-small'
      }
      return 'button is-danger is-outlined is-small'
    }
  },
  methods: {
    toggleState() {
      let currentToggle=this.item.is_hidden
      this.$store.commit('sceneList/toggleSceneList', {scene_id: this.item.scene_id, list: 'is_hidden'})
      this.item.is_hidden=!currentToggle
    }
  }
}
</script>
