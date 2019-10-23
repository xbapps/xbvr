<template>
  <div class="modal is-active">
    <div class="modal-background"></div>
    <div class="modal-card">
      <header class="modal-card-head">
        <p class="modal-card-title">Match file to scene</p>
        <button class="delete" @click="close" aria-label="close"></button>
      </header>
      <section class="modal-card-body">
        <div>
          <h6 class="title is-6">{{ file.filename }}</h6>
          <b-field label="Search">
            <div class="control">
              <input class="input" type="text" v-model='queryString' v-debounce:200ms="loadData" autofocus>
            </div>
          </b-field>
          <b-table
                  :data="data"
                  ref="table"
                  paginated
                  per-page="5">
            <template slot-scope="props">
              <b-table-column field="cover_url" label="Image" width="120">
                <vue-load-image>
                  <img slot="image" :src="getImageURL(props.row.cover_url)"/>
                  <img slot="preloader" src="/ui/images/blank.png"/>
                  <img slot="error" src="/ui/images/blank.png"/>
                </vue-load-image>
              </b-table-column>
              <b-table-column field="site" label="Site" sortable>
                {{ props.row.site }}
              </b-table-column>
              <b-table-column field="title" label="Title" sortable>
                <p v-if="props.row.title">{{ props.row.title }}</p>
                <small>
                  <b-tag rounded v-for="i in props.row.cast" :key="i.id">{{i.name}}</b-tag>
                </small>
              </b-table-column>
              <b-table-column field="release_date" label="Release date" sortable nowrap>
                {{format(parseISO(props.row.release_date), "yyyy-MM-dd")}}
              </b-table-column>
              <b-table-column field="scene_id" label="ID" sortable nowrap>
                {{props.row.scene_id}}
              </b-table-column>
              <b-table-column field="_score" label="Score" sortable>
                <b-progress show-value :value="props.row._score * 100"></b-progress>
              </b-table-column>
              <b-table-column field="_assign">
                  <button class="button" @click="assign(props.row.scene_id)">Assign</button>
              </b-table-column>
            </template>
          </b-table>
        </div>
      </section>
      <footer class="modal-card-foot">
        <!--        <button class="button is-small" @click="close">Cancel</button>-->
      </footer>
    </div>
  </div>
</template>

<script>
  import ky from "ky";
  import {format, parseISO} from "date-fns";
  import VueLoadImage from "vue-load-image";

  export default {
    name: "SceneMatch",
    components: {VueLoadImage,},
    data() {
      return {
        data: [],
        queryString: "",
        format, parseISO
      }
    },
    computed: {
      file() {
        return this.$store.state.overlay.match.file;
      },
    },
    mounted() {
      this.queryString = this.file.filename;
      this.loadData();
    },
    methods: {
      loadData: async function loadData() {
        let resp = await ky.get(`/api/scene/search`, {
          searchParams: {
            q: this.queryString,
          }
        }).json();

        this.data = resp.scenes;
      },
      getImageURL(u) {
        if (u.startsWith("http")) {
          return "/img/120x/" + u.replace("://", ":/");
        } else {
          return u;
        }
      },
      assign: async function assign(scene_id) {
        await ky.post(`/api/files/match`, {
          json: {
            file_id: this.toInt(this.$store.state.overlay.match.file.id),
            scene_id: scene_id,
          }
        });

        this.$store.dispatch("files/load");

        this.$store.commit("overlay/hideMatch");
      },
      close() {
        this.$store.commit("overlay/hideMatch");
      },
      toInt(value, radix, defaultValue) {
        return parseInt(value, radix || 10) || defaultValue || 0;
      },
    }
  }
</script>

<style scoped>
  .modal-card {
    width: 80%;
  }
</style>
