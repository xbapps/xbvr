<template>
  <div class="column">
    <b-loading :is-full-page="true" :active.sync="isLoading"></b-loading>

    <div class="columns is-multiline is-full">
      <div class="column">
        <strong>{{total}} results</strong>
      </div>
      <div class="column">
        <div class="is-pulled-right">
          <b-field>
            <span class="list-header-label">{{$t('Card size')}}</span>
            <b-radio-button v-model="cardSize" native-value="1" size="is-small">
              S
            </b-radio-button>
            <b-radio-button v-model="cardSize" native-value="2" size="is-small">
              M
            </b-radio-button>
            <b-radio-button v-model="cardSize" native-value="3" size="is-small">
              L
            </b-radio-button>
          </b-field>
        </div>
      </div>
    </div>

    <div class="columns is-multiline">
      <div :class="['column', 'is-multiline', cardSizeClass]"
           v-for="item in items" :key="item.id">
        <ActorCard :item="item"/>
      </div>
    </div>

    <div class="column is-full" v-if="items.length < total">
      <a class="button is-fullwidth" v-on:click="loadMore()">{{$t('Load more')}}</a>
    </div>
  </div>
</template>

<script>
  import ActorCard from "./ActorCard";

  export default {
    name: "List",
    components: {ActorCard},
    computed: {
      cardSize: {
        get() {
          return this.$store.state.actorList.filters.cardSize;
        },
        set(value) {
          this.$store.state.actorList.filters.cardSize = value;
        }
      },
      cardSizeClass() {
        switch (this.$store.state.actorList.filters.cardSize) {
          case "1":
            return "is-2";
          case "2":
            return "is-one-quarter";
          case "3":
            return "is-one-third";
          default:
            return "is-2";
        }
      },
      isLoading() {
        return this.$store.state.actorList.isLoading;
      },
      items() {
        return this.$store.state.actorList.items;
      },
      total() {
        return this.$store.state.actorList.total;
      }
    },
    methods: {
      async loadMore() {
        this.$store.dispatch("actorList/load", {offset: this.$store.state.actorList.offset});
      }
    }
  }
</script>

<style scoped>
  .list-header-label {
    padding-right: 1em;
  }
</style>
