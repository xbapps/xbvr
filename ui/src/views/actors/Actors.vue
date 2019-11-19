<template>
  <div class="container is-fluid">
    <div class="columns">

      <List/>

    </div>
  </div>
</template>

<script>
  import List from "./List";

  export default {
    name: "Actors",
    components: {List},
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
