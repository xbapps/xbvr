<template>
  <div class="modal is-active">
    <GlobalEvents
      @keyup.esc="close"
      @keydown.arrowLeft="playerStepBack"
      @keydown.arrowRight="playerStepForward"
      @keydown.o="prevScene"
      @keydown.p="nextScene"
      @keydown.f="$store.commit('sceneList/toggleSceneList', {scene_id: item.scene_id, list: 'favourite'})"
      @keydown.w="$store.commit('sceneList/toggleSceneList', {scene_id: item.scene_id, list: 'watchlist'})"
      @keydown.g="toggleGallery"
    />

    <div class="modal-background"></div>

    <div class="modal-card">
      <section class="modal-card-body">
        <div class="columns">

          <div class="column">
            <video ref="player"
                   width="640" height="640" class="video-js vjs-default-skin"
                   controls playsinline>
              <source :src="sourceUrl" type="video/mp4">
            </video>
          </div>

          <div class="column">

            <div class="block-info block">
              <div class="content">
                <h3>
                  <span v-if="item.title">{{item.title}}</span>
                  <span v-else class="missing">(no title)</span>
                  <small class="is-pulled-right">{{format(parseISO(item.release_date), "yyyy-MM-dd")}}</small>
                </h3>
                <small>{{item.site}}</small>
                <div class="columns">
                  <div class="column">
                    <star-rating :rating="item.star_rating" @rating-selected="setRating" :increment="0.5"
                                 :star-size="20"/>
                  </div>
                  <div class="column">
                    <div class="is-pulled-right">
                      <watchlist-button :item="item"/>&nbsp;
                      <favourite-button :item="item"/>
                    </div>
                  </div>
                </div>
              </div>
            </div>

            <div class="block-tags block">
              <b-taglist>
                <a v-for="(c, idx) in item.cast" :key="'cast' + idx" @click='showCastScenes([c.name])'
                   class="tag is-warning is-small">{{c.name}} ({{c.count}})</a>
                <a v-for="(tag, idx) in item.tags" :key="'tag' + idx" @click='showTagScenes([tag.name])'
                   class="tag is-info is-small">{{tag.name}} ({{tag.count}})</a>
              </b-taglist>
            </div>

            <div class="block-opts block">
              <b-tabs v-model="activeTab">

                <b-tab-item label="Gallery">
                  <div class="block-tab-content block">
                    <vue-gallery :images="galleryImages" :index="index" @close="index = null"
                                 :options="galleryOpts"></vue-gallery>
                    <div class="block is-pulled-left" v-for="(i, idx) in images" :key="idx">
                      <vue-load-image>
                        <img slot="image" :src="getImageURL(i.url, '120x')" @click="index = idx"/>
                        <img slot="preloader" src="/ui/images/blank.png"/>
                        <img slot="error" src="/ui/images/blank.png"/>
                      </vue-load-image>
                    </div>
                  </div>
                </b-tab-item>

                <b-tab-item :label="`Cuepoints (${sortedCuepoints.length})`">
                  <div class="block-tab-content block">
                    <div class="block">
                      <b-field grouped>
                        <b-select v-model="tagPosition">
                          <option v-for="(option, idx) in cuepointPositionTags" :value="option" :key="idx">
                            {{ option }}
                          </option>
                        </b-select>
                        <b-select v-model="tagAct">
                          <option v-for="(option, idx) in cuepointActTags" :value="option" :key="idx">
                            {{ option }}
                          </option>
                        </b-select>
                        <b-button @click="addCuepoint">Add cuepoint</b-button>
                      </b-field>
                    </div>
                    <div class="content is-small">
                      <ul>
                        <li v-for="c in sortedCuepoints">
                          <code>{{humanizeSeconds(c.time_start)}}</code> -
                          <a @click="playCuepoint(c)"><strong>{{c.name}}</strong></a>
                        </li>
                      </ul>
                    </div>
                  </div>
                </b-tab-item>

                <b-tab-item :label="`Files (${fileCount})`">
                  <div class="block-tab-content block">
                    <div class="content media is-small" v-for="f in item.file">
                      <div class="media-left">
                        <button rounded class="button is-success is-small" @click='playFile(f)'>
                          <b-icon pack="fas" icon="play" size="is-small"></b-icon>
                        </button>
                      </div>
                      <div class="media-content" style="overflow-wrap: break-word;">
                        <strong>{{f.filename}}</strong><br/>
                        <small>{{prettyBytes(f.size)}}, {{f.video_width}}x{{f.video_height}},
                          {{format(parseISO(f.created_time), "yyyy-MM-dd")}}</small>
                      </div>
                      <div class="media-right">
                        <button class="button is-danger is-small is-outlined" @click='removeFile(f)'>
                          <b-icon pack="fas" icon="trash" size="is-small"></b-icon>
                        </button>
                      </div>
                    </div>
                  </div>
                </b-tab-item>

                <b-tab-item label="Watch history">
                  <div class="block-tab-content block">
                    <div>
                      {{historySessionsCount}} view sessions, total duration
                      {{humanizeSeconds(historySessionsDuration)}}
                    </div>
                    <div class="content is-small">
                      <div class="block" v-for="session in item.history">
                        <strong>{{format(parseISO(session.time_start), "yyyy-MM-dd kk:mm:ss")}} -
                          {{humanizeSeconds(session.duration)}}</strong>
                      </div>
                    </div>
                  </div>
                </b-tab-item>

              </b-tabs>
            </div>

          </div>
        </div>
      </section>
    </div>
    <button class="modal-close is-large" aria-label="close"
            @click="close()"></button>
    <a class="prev" @click="prevScene" v-if="$store.getters['sceneList/prevScene'](item) !== null"
       title="Keyboard shortcut: O">&#10094;</a>
    <a class="next" @click="nextScene" v-if="$store.getters['sceneList/nextScene'](item) !== null"
       title="Keyboard shortcut: P">&#10095;</a>
  </div>
</template>

<script>
  import ky from "ky";
  import videojs from "video.js";
  import {format, formatDistance, parseISO} from "date-fns";
  import prettyBytes from "pretty-bytes";
  import VueLoadImage from "vue-load-image";
  import VueGallery from 'vue-gallery';
  import GlobalEvents from 'vue-global-events';
  import StarRating from 'vue-star-rating';
  import FavouriteButton from "../../components/FavouriteButton";
  import WatchlistButton from "../../components/WatchlistButton";

  export default {
    name: "Details",
    components: {VueLoadImage, VueGallery, GlobalEvents, StarRating, WatchlistButton, FavouriteButton},
    data() {
      return {
        index: 1,
        activeTab: 0,
        player: {},
        tagAct: "",
        tagPosition: "",
        cuepointPositionTags: ["", "standing", "sitting", "laying", "kneeling"],
        cuepointActTags: ["", "handjob", "blowjob", "doggy", "cowgirl", "revcowgirl", "missionary", "titfuck", "anal", "cumshot", "69", "facesit"],
        galleryOpts: {
          // container: this.$refs.container,
        }
      }
    },
    computed: {
      item() {
        return this.$store.state.overlay.details.scene;
      },
      // Properties for VideoJS player
      sourceUrl() {
        if (this.$store.state.overlay.details.scene.is_available) {
          return "/api/dms/file/" + this.$store.state.overlay.details.scene.file[0].id + "?dnt=1";
        }
        return "";
      },
      // Properties for gallery
      images() {
        return JSON.parse(this.item.images);
      },
      galleryImages() {
        return this.images.map((e) => {
          return "/img/" + e.url.replace("://", ":/");
        })
      },
      // Tab: cuepoints
      sortedCuepoints() {
        if (this.item.cuepoints !== null) {
          return this.item.cuepoints.sort((a, b) => (a.time_start > b.time_start) ? 1 : -1);
        }
        return [];
      },
      // Tab: files
      fileCount() {
        if (this.item.file !== null) {
          return this.item.file.length;
        }
        return 0;
      },
      // Tab: history
      historySessionsCount() {
        if (this.item.history !== null) {
          return this.item.history.length;
        }
        return 0;
      },
      historySessionsDuration() {
        if (this.item.history !== null) {
          let total = 0;
          this.item.history.map(i => {
            total = total + i.duration;
          });
          return total;
        }
        return 0;
      },
    },
    mounted() {
      this.setupPlayer();
    },
    methods: {
      setupPlayer() {
        this.player = videojs(this.$refs.player);

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
                this.$store.commit("overlay/hideDetails");
              }
            }
          }
        });

        this.updatePlayer();
      },
      updatePlayer() {
        this.index = null;
        this.player.reset();

        let vr = this.player.vr({
          projection: '360',
          forceCardboard: false
        });

        this.player.on("loadedmetadata", function () {
          vr.camera.position.set(-1, 0, -1);
        });

        this.player.src({src: this.sourceUrl, type: "video/mp4"});
        this.player.poster(this.getImageURL(this.item.cover_url, ""));
      },
      showCastScenes(actor) {
        this.$store.state.sceneList.filters.cast = actor;
        this.$store.state.sceneList.filters.sites = [];
        this.$store.state.sceneList.filters.tags = [];
        this.$router.push({
          name: 'scenes',
          query: {q: this.$store.getters['sceneList/filterQueryParams']}
        });
        this.close();
      },
      showTagScenes(tag) {
        this.$store.state.sceneList.filters.cast = [];
        this.$store.state.sceneList.filters.sites = [];
        this.$store.state.sceneList.filters.tags = tag;
        this.$router.push({
          name: 'scenes',
          query: {q: this.$store.getters['sceneList/filterQueryParams']}
        });
        this.close();
      },
      playFile(file) {
        this.player.src({type: "video/mp4", src: "/api/dms/file/" + file.id + "?dnt=1"});
        this.player.play();
      },
      removeFile(file) {
        this.$buefy.dialog.confirm({
          title: 'Remove file',
          message: `You're about to remove file <strong>${file.filename}</strong> from <strong>disk</strong>.`,
          type: 'is-danger',
          hasIcon: true,
          onConfirm: () => {
            ky.delete(`/api/files/file/${file.id}`).json().then(data => {
              this.$store.commit("overlay/showDetails", {scene: data});
            });
          }
        });
      },
      getImageURL(u, size) {
        if (u.startsWith("http")) {
          return "/img/" + size + "/" + u.replace("://", ":/");
        } else {
          return u;
        }
      },
      playCuepoint(cuepoint) {
        this.player.currentTime(cuepoint.time_start);
        this.player.play();
      },
      addCuepoint() {
        let name = "";
        if (this.tagAct !== "") {
          name = this.tagAct;
        }
        if (this.tagPosition !== "") {
          name = this.tagPosition;
        }
        if (this.tagPosition !== "" && this.tagAct !== "") {
          name = `${this.tagPosition}-${this.tagAct}`;
        }
        ky.post(`/api/scene/cuepoint/${this.item.id}`, {
          json: {
            name: name,
            time_start: this.player.currentTime()
          }
        }).json().then(data => {
          this.$store.commit("overlay/showDetails", {scene: data});
        });
      },
      close() {
        this.player.dispose();
        this.$store.commit("overlay/hideDetails");
      },
      humanizeSeconds(seconds) {
        return new Date(seconds * 1000).toISOString().substr(11, 8);
      },
      setRating(val) {
        ky.post(`/api/scene/rate/${this.item.id}`, {json: {rating: val}});

        let updatedScene = Object.assign({}, this.item);
        updatedScene.star_rating = val;
        this.$store.commit('sceneList/updateScene', updatedScene);
      },
      nextScene() {
        let data = this.$store.getters['sceneList/nextScene'](this.item);
        if (data !== null) {
          this.$store.commit("overlay/showDetails", {scene: data});
          this.updatePlayer();
        }
      },
      prevScene() {
        let data = this.$store.getters['sceneList/prevScene'](this.item);
        if (data !== null) {
          this.$store.commit("overlay/showDetails", {scene: data});
          this.updatePlayer();
        }
      },
      playerStepBack() {
        let wasPlaying = !this.player.paused();
        if (wasPlaying) {
          this.player.pause();
        }
        let seekTime = this.player.currentTime() - 5;
        if (seekTime <= 0) {
          seekTime = 0;
        }
        this.player.currentTime(seekTime);
        if (wasPlaying) {
          this.player.play();
        }
      },
      playerStepForward() {
        let duration = this.player.duration();
        let wasPlaying = !this.player.paused();
        if (wasPlaying) {
          this.player.pause();
        }
        let seekTime = this.player.currentTime() + 5;
        if (seekTime >= duration) {
          seekTime = wasPlaying ? duration - .001 : duration;
        }
        this.player.currentTime(seekTime);
        if (wasPlaying) {
          this.player.play();
        }
      },
      toggleGallery() {
        if (this.index === null) {
          this.index = 1;
        } else {
          this.index = null;
        }
      },
      format,
      parseISO,
      prettyBytes,
      formatDistance,
    }
  }
</script>

<style lang="less" scoped>
  .video-js {
    margin: 0 auto;
  }

  .modal-card {
    width: 85%;
  }

  .missing {
    opacity: 0.6;
  }

  .block-tab-content {
    flex: 1 1 auto;
  }

  .block-info {
  }

  .block-tags {
    max-height: 200px;
    overflow: scroll;
    scrollbar-width: none;
  }

  .block-tags::-webkit-scrollbar {
    display: none;
  }

  .block-opts {
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
</style>
