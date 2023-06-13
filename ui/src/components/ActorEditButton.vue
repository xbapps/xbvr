<template>
  <a class="button is-dark is-outlined is-small"
    @click="editActor(actor)"
    title="Edit Actor details">
    <b-icon pack="mdi" icon="lead-pencil" size="is-small" />
  </a>
</template>

<script>
export default {
  name: 'ActorEditButton',
  props: { actor: Object },
  methods: {
    editActor (actor) {
      this.$store.commit('overlay/editActorDetails', { actor })
    }
  }
}
</script>
