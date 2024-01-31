<template>
  <a class="button is-dark is-outlined is-small"
    @click="editScene(item)"    
    :title="item.id == 0 ? 'Display scene details': 'Edit scene details'">
    <b-icon pack="mdi" icon="lead-pencil" size="is-small" />
  </a>
</template>

<script>
export default {
  name: 'EditButton',
  props: { item: Object },
  methods: {
    editScene (scene) {
      this.$store.commit('overlay/editDetails', { scene })
    }
  }
}
</script>
