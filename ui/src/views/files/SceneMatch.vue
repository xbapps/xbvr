<template>
  <div class="modal is-active">
    <div class="modal-background"></div>
    <div class="modal-card">
      <section class="modal-card-body">
        <div>
          <div class="columns is-vcentered">
            <div class="column is-narrow">
              <video ref="player"
                width="384" height="216" class="video-js vjs-default-skin" controls playsinline>
              </video>
            </div>
            <div class="column">
              <div class="is-size-4 has-text-weight-bold filename">{{file.filename}}</div>
              <div>
                <b-field grouped group-multiline>
                  <div class="control">
                    <b-taglist attached>
                      <b-tag type="is-dark" rounded>Location</b-tag>
                      <b-tag type="is-info" rounded>{{file.path}}</b-tag>
                    </b-taglist>
                  </div>
                  <div class="control">
                    <b-taglist attached>
                      <b-tag type="is-dark" rounded>Size</b-tag>
                      <b-tag type="is-info" rounded>{{prettyBytes(file.size)}}</b-tag>
                    </b-taglist>
                  </div>
                  <div class="control">
                    <b-taglist attached>
                      <b-tag type="is-dark" rounded>Created</b-tag>
                      <b-tag type="is-info" rounded>
                        {{format(parseISO(file.created_time), "yyyy-MM-dd hh:mm:ss")}}
                      </b-tag>
                    </b-taglist>
                  </div>
                  <div class="control">
                    <b-taglist attached>
                      <b-tag type="is-dark" rounded>Duration</b-tag>
                      <b-tag v-if="file.duration !== 0" type="is-info" rounded>
                        {{humanizeSeconds(file.duration)}}
                      </b-tag>
                      <b-tag v-else type="is-info" rounded>-</b-tag>
                    </b-taglist>
                  </div>
                  <div class="control">
                    <b-taglist attached>
                      <b-tag type="is-dark" rounded>Resolution</b-tag>
                      <b-tag v-if="file.video_width !== 0 || file.video_height !== 0" type="is-info" rounded>
                        {{file.video_width}} × {{file.video_height}}
                      </b-tag>
                      <b-tag v-else type="is-info" rounded>-</b-tag>
                    </b-taglist>
                  </div>
                  <div class="control">
                    <b-taglist attached>
                      <b-tag type="is-dark" rounded>Bitrate</b-tag>
                      <b-tag v-if="file.video_bitrate !== 0" type="is-info" rounded>
                        {{prettyBytes(file.video_bitrate)}}
                      </b-tag>
                      <b-tag v-else type="is-info" rounded>-</b-tag>
                    </b-taglist>
                  </div>
                  <div class="control">
                    <b-taglist attached>
                      <b-tag type="is-dark" rounded>Framerate</b-tag>
                      <b-tag v-if="file.video_avgfps_val !== 0" type="is-info" rounded>
                        {{file.video_avgfps_val}}
                      </b-tag>
                      <b-tag v-else type="is-info" rounded>-</b-tag>
                    </b-taglist>
                  </div>
                </b-field>
              </div>
              <div class="searchbox">
                <b-field :label="$t('Search')">
                  <b-input v-model='queryString' v-debounce:200ms="loadData" autofocus></b-input>
                </b-field>
              </div>
            </div>
          </div>
          <div>
            <b-table :data="data" ref="table" paginated :current-page.sync="currentPage" per-page="5">
              <template slot-scope="props">
                <b-table-column field="cover_url" :label="$t('Image')" width="120">
                  <vue-load-image>
                    <img slot="image" :src="getImageURL(props.row.cover_url)"/>
                    <img slot="preloader" src="/ui/images/blank.png"/>
                    <img slot="error" src="/ui/images/blank.png"/>
                  </vue-load-image>
                </b-table-column>
                <b-table-column field="site" :label="$t('Site')" sortable>
                  {{props.row.site}}
                </b-table-column>
                <b-table-column field="title" :label="$t('Title')" sortable>
                  <p v-if="props.row.title">{{props.row.title}}</p>
                  <small>
                    <b-tag rounded v-for="i in props.row.cast" :key="i.id">{{i.name}}</b-tag>
                  </small>
                </b-table-column>
                <b-table-column field="release_date" :label="$t('Release date')" sortable nowrap>
                  {{format(parseISO(props.row.release_date), "yyyy-MM-dd")}}
                </b-table-column>
                <b-table-column field="scene_id" :label="$t('ID')" sortable nowrap>
                  {{props.row.scene_id}}
                </b-table-column>
                <b-table-column field="_score" :label="$t('Score')" sortable>
                  <b-progress show-value :value="props.row._score * 100"></b-progress>
                </b-table-column>
                <b-table-column field="_assign">
                  <button class="button" @click="assign(props.row.scene_id)">{{$t("Assign")}}</button>
                </b-table-column>
              </template>
            </b-table>
          </div>
        </div>
      </section>
    </div>
    <button class="modal-close is-large" aria-label="close" @click="close()"></button>
    <a class="prev" @click="prevFile">&#10094;</a>
    <a class="next" @click="nextFile">&#10095;</a>    
  </div>
</template>

<script>
  import ky from "ky";
  import {format, parseISO} from "date-fns";
  import VueLoadImage from "vue-load-image";
  import videojs from "video.js";
  import vr from "videojs-vr";
  import hotkeys from "videojs-hotkeys";
  import prettyBytes from "pretty-bytes";

  export default {
    name: "SceneMatch",
    components: {VueLoadImage,},
    data() {
      return {
        data: [],
        prettyBytes,
        format,
        parseISO,
        player: {},
        currentPage: 1,
        queryString: "",
        format, parseISO
      }
    },
    computed: {
      file() {
        return this.$store.state.overlay.match.file;
      }
    },
    mounted() {
      this.player = videojs(this.$refs.player);
        let vr = this.player.vr({
          projection: '360',
          forceCardboard: false
        });
        this.player.hotkeys({
          alwaysCaptureHotkeys: true,
          volumeStep: 0.1,
          seekStep: 5,
          enableModifiersForNumbers: false,
          customKeys: {
            closeModal: {
              key: function (event) {
                return event.which === 27
              },
              handler: (player, options, event) => {
                this.player.dispose();
                this.$store.commit("overlay/hideMatch");
              }
            }
          }
        });
        this.player.on("loadedmetadata", function () {
          vr.camera.position.set(-1, 0, -1);
        });
      this.initView();
    },
    methods: {
      initView() {
        this.data = [];
        this.queryString = this.file.filename.replace(/\./g, " ").replace(/\_/g, " ").replace(/\+/g, " ").replace(/\-/g, " ");
        this.player.src({type: 'video/mp4', src: "/api/dms/file/" + this.file.id + "?dnt=1"});
        this.loadData();
      },
      loadData: async function loadData() {
        let resp = await ky.get(`/api/scene/search`, {
          searchParams: {
            q: this.queryString,
          }
        }).json();

        this.data = resp.scenes;
        this.currentPage = 1;
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

        let data = this.$store.getters['files/nextFile'](this.file);
        if (data !== null) {
          this.nextFile();
        }
      },
      nextFile() {
        let data = this.$store.getters['files/nextFile'](this.file);
        if (data !== null) {
          this.$store.commit("overlay/showMatch", {file: data});
          this.initView();
        }
      },
      prevFile() {
        let data = this.$store.getters['files/prevFile'](this.file);
        if (data !== null) {
          this.$store.commit("overlay/showMatch", {file: data});
          this.initView();
        }
      },
      close() {
        this.player.dispose();
        this.$store.commit("overlay/hideMatch");
      },
      toInt(value, radix, defaultValue) {
        return parseInt(value, radix || 10) || defaultValue || 0;
      },
      humanizeSeconds(seconds) {
        return new Date(seconds * 1000).toISOString().substr(11, 8);
      }
    }
  }
</script>

<style scoped>
  .modal-card {
    position: absolute;
    top: 4em;
    width: 85%;
  }

  .filename {
    padding-bottom: 0.6em;
  }

  .searchbox {
    padding-top: 1em;
    width: 80%;
  }

  .prev, .next {
    cursor: pointer;
    position: absolute;
    top: 50%;
    width: auto;
    padding: 16px;
    margin-top: -50px;
    color: white;
    font-weight: bold;
    font-size: 24px;
    border-radius: 0 3px 3px 0;
    user-select: none;
    -webkit-user-select: none;
  }

  .next {
    right: 0;
    border-radius: 3px 0 0 3px;
  }

  .prev {
    left: 0;
    border-radius: 3px 0 0 3px;
  }
  
  .video-js {
    margin: 0 auto;
  }
</style>
