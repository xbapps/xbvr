<template>
  <div>
    <div class="container is-fluid">
      <div>
        <Filters/>
      </div>
      <div class="columns">
        <div class="column">
          <List/>
        </div>
      </div>
    </div>
    <SceneMatch v-if="showMatchOverlay"/>
    <Details v-if="showDetailsOverlay"/>
  </div>
</template>

<script>
  import Filters from "./Filters";
  import List from "./List";
  import SceneMatch from "./SceneMatch";
  import Details from "../scenes/Details";

  export default {
    name: "Files",
    components: {Filters, List, SceneMatch, Details},
    computed: {
      showMatchOverlay() {
        return this.$store.state.overlay.match.show;
      },
      showDetailsOverlay() {
        return this.$store.state.overlay.details.show;
      },
    }
  }
</script>
