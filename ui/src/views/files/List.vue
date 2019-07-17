<template>
  <div>
    <div class="columns">
      <div class="column">
        <div v-if="files.length > 0">
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
            <tr v-for="v in files" v-bind:key="v.id">
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
      </div>
    </div>
  </div>
</template>

<script>
  import ky from "ky";
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
    mounted() {
      this.getData();
    },
    methods: {
      getData: async function getData() {
        this.files = await ky.get(`/api/files/list/unmatched`).json();
      },
      play(file) {
        this.$store.commit("overlay/showPlayer", {file: file});
      },
      match(file) {
        this.$store.commit("overlay/showMatch", {file: file});
      }
    },
    computed: {
    }
  }
</script>
