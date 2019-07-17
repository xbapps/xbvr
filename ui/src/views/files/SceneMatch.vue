<template>
  <div class="modal is-active">
    <div class="modal-background"></div>
    <div class="modal-card">
      <header class="modal-card-head">
        <p class="modal-card-title">Match file to scene</p>
        <button class="delete" @click="close" aria-label="close"></button>
      </header>
      <section class="modal-card-body">
        <p style="text-wrap: initial">{{file.filename}}</p>
        <div>
          <b-field label="Search">
            <div class="control">
              <input class="input" type="text" v-model='queryString' v-debounce:200ms="loadData" autofocus>
            </div>
          </b-field>
          <b-table
                  :data="data"
                  ref="table"
                  paginated
                  per-page="5"
                  detailed
                  detail-key="cover_url"
                  :show-detail-icon="true">
            <template slot-scope="props">
              <b-table-column field="cover_url" label="Image" width="120">
                <img :src="getImageURL(props.row.cover_url)"/>
              </b-table-column>
              <b-table-column field="site" label="Site">
                {{ props.row.site }}
              </b-table-column>
              <b-table-column field="title" label="Title">
                {{ props.row.title }}
              </b-table-column>
              <b-table-column field="release_date" label="Release date">
                {{format(parse(props.row.release_date), "YYYY-MM-DD")}}
              </b-table-column>
            </template>
            <template slot="detail" slot-scope="props">
              <article class="media">
                <button class="button" @click="assign(props.row.scene_id)">Assign file with this scene</button>
              </article>
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
  import {format, parse} from "date-fns";

  export default {
    name: "SceneMatch",
    data() {
      return {
        data: [],
        queryString: "",
        format, parse
      }
    },
    computed: {
      file() {
        return this.$store.state.overlay.match.file;
      },
    },
    mounted() {
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

</style>