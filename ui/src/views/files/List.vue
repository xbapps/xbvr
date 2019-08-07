<template>
  <div>
    <div class="columns">
      <div class="column">
        <b-loading :is-full-page="true" :active.sync="isLoading"></b-loading>

        <div v-if="items.length > 0 && !isLoading">
          <b-table :data="items" ref="table">
            <template slot-scope="props">
              <b-table-column field="filename" label="File" sortable>
                {{props.row.filename}}
              </b-table-column>
              <b-table-column field="path" label="Folder" sortable>
                {{props.row.path}}
              </b-table-column>
              <b-table-column field="size" label="Size" sortable style="white-space: nowrap;">
                {{prettyBytes(props.row.size)}}
              </b-table-column>
              <b-table-column field="video_height" label="Resolution" sortable>
                {{props.row.video_width}}x{{props.row.video_height}}
              </b-table-column>
              <b-table-column style="white-space: nowrap;">
                <b-button @click="play(props.row)">Play</b-button>&nbsp;
                <b-button @click="match(props.row)">Match to scene</b-button>
              </b-table-column>
            </template>
          </b-table>
        </div>
        <div v-if="items.length === 0 && !isLoading">
          <section class="hero is-large">
            <div class="hero-body">
              <div class="container has-text-centered">
                <h1 class="title">
                  <span class="icon">
                    <i class="far fa-check-circle is-superlarge"></i>
                  </span>
                </h1>
                <h2 class="subtitle">
                  All of your files are linked to scenes
                </h2>
              </div>
            </div>
          </section>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
  import prettyBytes from "pretty-bytes";
  import {distanceInWordsToNow, parse} from "date-fns";
  import BButton from "buefy/src/components/button/Button";

  export default {
    name: "List",
    components: {BButton},
    data() {
      return {
        files: [],
        prettyBytes,
        parse,
        distanceInWordsToNow,
      }
    },
    computed: {
      isLoading() {
        return this.$store.state.files.isLoading;
      },
      items() {
        return this.$store.state.files.items;
      },
    },
    mounted() {
      this.$store.dispatch("files/load");
    },
    methods: {
      play(file) {
        this.$store.commit("overlay/showPlayer", {file: file});
      },
      match(file) {
        this.$store.commit("overlay/showMatch", {file: file});
      }
    },
  }
</script>

<style scoped>
  .is-superlarge {
    height: 96px;
    max-height: 96px;
    max-width: 96px;
    min-height: 96px;
    min-width: 96px;
    width: 96px;
  }
</style>