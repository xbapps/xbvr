<template>
  <div class="container is-fluid">
    <div class="columns">

      <div class="column is-2">
        <Filters/>
      </div>
      <List/>

    </div>
  </div>
</template>

<script>
  import Filters from "./Filters";
  import List from "./List";

  export default {
    name: "Actors",
    components: {Filters, List},
    beforeRouteEnter(to, from, next) {
      next(vm => {
        vm.$store.dispatch("actorList/load", {offset: 0});
      });
    },
    beforeRouteUpdate(to, from, next) {
      this.$store.dispatch("actorList/load", {offset: 0});
      next();
    },
  }
</script>
