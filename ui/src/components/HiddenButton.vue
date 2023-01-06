<template>
  <a :class="buttonClass"
     @click="$store.commit('sceneList/toggleSceneList', {scene_id: item.scene_id, list: 'is_hidden'})"
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
  }
}
</script>
