<template>
  <a :class="buttonClass"
     @click="toggleState()"
     :title="actor.watchlist ? 'Remove from watchlist' : 'Add to watchlist'">
    <b-icon pack="mdi" :icon="actor.watchlist ? 'calendar-check' : 'calendar-blank'" size="is-small"/>
  </a>
</template>

<script>
export default {
  name: 'WatchlistButton',
  props: { actor: Object },
  computed: {
    buttonClass () {
      if (this.actor.watchlist) {
        return 'button is-primary is-small'
      }
      return 'button is-primary is-outlined is-small'
    }
  },
  methods: {
    toggleState() {
      let currentToggle=this.actor.watchlist
      this.$store.commit('actorList/toggleActorList', {actor_id: this.actor.id, list: 'watchlist'})
      this.actor.watchlist=!currentToggle
    }
  }
}
</script>
