<template>
  <a :class="buttonClass"
     @click="$store.commit('sceneList/toggleSceneList', {scene_id: item.scene_id, list: 'versions'})">
    <b-icon pack="mdi" :icon="item.versions ? 'image-multiple' : 'image-multiple-outline'" size="is-small"/>
  </a>
</template>

<script>
  export default {
    name: "VersionsButton",
    props: {item: Object},
    computed: {
      buttonClass() {
        if (this.item.versions) {
          return "button is-info is-small";
        }
        return "button is-info is-outlined is-small";
      },
    }
  }
</script>
