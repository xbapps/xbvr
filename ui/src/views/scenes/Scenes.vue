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
    computed: {
      showOverlay() {
        return this.$store.state.detailsOverlay.show;
      }
    }
  }
</script>
