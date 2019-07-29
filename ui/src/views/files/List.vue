<template>
  <div>
    <div class="columns">
      <div class="column">
        <div v-if="items.length > 0">
          <table class="table">
            <thead>
            <tr>
              <th>File</th>
              <th>Folder</th>
              <th>Size</th>
              <th>Resolution</th>
              <th></th>
            </tr>
            </thead>
            <tbody>
            <tr v-for="v in items" v-bind:key="v.id">
              <td>{{v.filename}}</td>
              <td>{{v.path}}</td>
              <td nowrap>{{prettyBytes(v.size)}}</td>
              <td>{{v.video_width}}x{{v.video_height}}</td>
              <td nowrap>
                <b-button @click="play(v)">Play</b-button>&nbsp;
                <b-button @click="match(v)">Match to scene</b-button>
              </td>
            </tr>
            </tbody>
          </table>
        </div>
        <div v-else>
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