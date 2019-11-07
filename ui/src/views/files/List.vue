<template>
  <div>
    <div class="columns">
      <div class="column">
        <b-loading :is-full-page="true" :active.sync="isLoading"></b-loading>

        <div v-if="items.length > 0 && !isLoading">
          <b-table :data="items" ref="table">
            <template slot-scope="props">
              <b-table-column field="filename" :label="$t('File')">
                {{props.row.filename}}
                <br/><small>{{props.row.path}}</small>
              </b-table-column>
              <b-table-column field="created_time" :label="$t('Created')" style="white-space: nowrap;">
                {{format(parseISO(props.row.created_time), "yyyy-MM-dd hh:mm:ss")}}
              </b-table-column>
              <b-table-column field="size" :label="$t('Size')" style="white-space: nowrap;">
                {{prettyBytes(props.row.size)}}
              </b-table-column>
              <b-table-column field="video_height" :label="$t('Resolution')">
                {{props.row.video_width}}x{{props.row.video_height}}
              </b-table-column>
              <b-table-column style="white-space: nowrap;">
                <b-button @click="play(props.row)">{{$t('Play')}}</b-button>&nbsp;
                <b-button @click="match(props.row)">{{$t('Match to scene')}}</b-button>
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
                  {{$t('All of your files are linked to scenes')}}
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
  import {format, parseISO} from "date-fns";

  export default {
    name: "List",
    data() {
      return {
        files: [],
        prettyBytes,
        format,
        parseISO,
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
  small {
    opacity: 0.6;
  }

  .is-superlarge {
    height: 96px;
    max-height: 96px;
    max-width: 96px;
    min-height: 96px;
    min-width: 96px;
    width: 96px;
  }
</style>
