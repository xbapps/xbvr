<template>
  <a :class="buttonClass"
     @click="$store.commit('sceneList/toggleSceneList', {scene_id: item.scene_id, list: 'multifiles'})">
    <b-icon pack="mdi" :icon="item.multifiles ? 'image-multiple' : 'image-multiple-outline'" size="is-small"/>
  </a>
</template>

<script>
  export default {
    name: "MultiFilesButton",
    props: {item: Object},
    computed: {
      buttonClass() {
        if (this.item.multifiles) {
          return "button is-primary is-small";
        }
        return "button is-primary is-outlined is-small";
      },
    }
  }
</script>
