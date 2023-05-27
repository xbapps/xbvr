<template>
  <a :class="buttonClass"
     @click="toggleState()"
     :title="actor.favourite ? 'Remove from favourites' : 'Add to favourites'">
    <b-icon pack="mdi" :icon="actor.favourite ? 'heart' : 'heart-outline'" size="is-small"/>
  </a>
</template>

<script>
export default {
  name: 'ActorFavouriteButton',
  props: { actor: Object },
  computed: {
    buttonClass () {
      if (this.actor.favourite) {
        return 'button is-danger is-small'
      }
      return 'button is-danger is-outlined is-small'
    }
  },
  methods: {
    toggleState() {
      let currentToggle=this.actor.favourite
      this.$store.commit('actorList/toggleActorList', {actor_id: this.actor.id, list: 'favourite'})
      this.actor.favourite=!currentToggle
    }
  }
}
</script>
