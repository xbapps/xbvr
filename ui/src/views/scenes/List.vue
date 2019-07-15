<template>
  <div class="column">
    <div class="columns is-multiline is-full">
      <div class="column">
        <strong>{{total}} results</strong>
      </div>
    </div>

    <div class="columns is-multiline">
      <div :class="['column', 'is-multiline', cardSizeClass]"
           v-for="item in items" :key="item.id">
        <SceneCard :item="item"/>
      </div>
    </div>

    <div class="column is-full" v-if="items.length < total">
      <a class="button is-fullwidth" v-on:click="loadMore()">Load more</a>
    </div>
  </div>
</template>

<script>
  import SceneCard from "./SceneCard";

  export default {
    name: "List",
    components: {SceneCard},
    mounted() {
      this.$store.dispatch("loadList", {offset: 0});
    },
    computed: {
      cardSizeClass() {
        switch (this.$store.state.sceneList.filters.cardSize) {
          case "1":
            return "is-one-fifth";
          case "2":
            return "is-one-quarter";
          case "3":
            return "is-one-third";
          default:
            return "is-one-fifth";
        }
      },
      items() {
        return this.$store.state.sceneList.items;
      },
      total() {
        return this.$store.state.sceneList.total;
      }
    },
    methods: {
      async loadMore() {
        this.$store.dispatch("loadList", {offset: this.$store.state.sceneList.offset});
      }
    }
  }
</script>
