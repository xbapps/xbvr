<template>
  <div class="container is-fluid">
    <div class="columns">

      <div class="column is-one-fifth">
        <Filters/>
      </div>

      <List/>
      <Details v-if="showOverlay"/>

    </div>
  </div>
</template>

<script>
  import Filters from "./Filters";
  import List from "./List";
  import Details from "./Details";

  export default {
    name: "Scenes",
    components: {Filters, List, Details},
    beforeRouteEnter(to, from, next) {
      next(vm => {
        if (to.query !== undefined) {
          vm.$store.commit('sceneList/stateFromQuery', to.query);
        }
        vm.$store.dispatch("sceneList/load", {offset: 0});
      });
    },
    beforeRouteUpdate(to, from, next) {
      if (to.query !== undefined) {
        this.$store.commit('sceneList/stateFromQuery', to.query);
      }
      this.$store.dispatch("sceneList/load", {offset: 0});
      next();
    },
    computed: {
      showOverlay() {
        return this.$store.state.overlay.details.show;
      }
    }
  }
</script>
