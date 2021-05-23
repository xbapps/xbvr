<template>
  <div class="modal is-active">
    <GlobalEvents
      :filter="e => !['INPUT', 'TEXTAREA'].includes(e.target.tagName)"
      @keyup.esc="close"
      @keydown.arrowLeft="playerStepBack"
      @keydown.arrowRight="playerStepForward"
      @keydown.o="prevScene"
      @keydown.p="nextScene"
      @keydown.f="$store.commit('sceneList/toggleSceneList', {scene_id: item.scene_id, list: 'favourite'})"
      @keydown.w="$store.commit('sceneList/toggleSceneList', {scene_id: item.scene_id, list: 'watchlist'})"
      @keydown.e="$store.commit('overlay/editDetails', {scene: item.scene})"
      @keydown.g="toggleGallery"
    />

    <div class="modal-background"></div>

    <div class="modal-card">
      <section class="modal-card-body">
        <div class="columns">

          <div class="column">
            <b-tabs v-model="activeMedia" position="is-centered" :animated="false">

              <b-tab-item label="Gallery">
                <b-carousel v-model="carouselSlide" :autoplay="false" :indicator-inside="false">
                  <b-carousel-item v-for="(carousel, i) in images" :key="i">
                    <div class="image is-1by1 is-full"
                         v-bind:style="{backgroundImage: `url(${getImageURL(carousel.url, '700,fit')})`, backgroundSize: 'contain', backgroundPosition: 'center', backgroundRepeat: 'no-repeat'}"></div>
                  </b-carousel-item>
                  <template slot="indicators" slot-scope="props">
                      <span class="al image">
                        <vue-load-image>
                          <img slot="image" :src="getIndicatorURL(props.i)" style="height:40px;"/>
                          <img slot="preloader" src="/ui/images/blank.png" style="height:40px;"/>
                          <img slot="error" src="/ui/images/blank.png" style="height:40px;"/>
                        </vue-load-image>
                      </span>
                  </template>
                </b-carousel>
              </b-tab-item>

              <b-tab-item label="Player">
                <video ref="player" class="video-js vjs-default-skin" controls playsinline preload="none"/>
              </b-tab-item>

            </b-tabs>

          </div>

          <div class="column">

            <div class="block-info block">
              <div class="content">
                <h3>
                  <span v-if="item.title">{{ item.title }}</span>
                  <span v-else class="missing">(no title)</span>
                  <small class="is-pulled-right">{{ format(parseISO(item.release_date), "yyyy-MM-dd") }}</small>
                </h3>
                <small>{{ item.site }}</small>
                <div class="columns">
                  <div class="column">
                    <star-rating :key="item.id" :rating="item.star_rating" @rating-selected="setRating"
                                 :increment="0.5" :star-size="20"/>
                  </div>
                  <div class="column">
                    <div class="is-pulled-right">
                      <watchlist-button :item="item"/>&nbsp;
                      <favourite-button :item="item"/>&nbsp;
                      <edit-button :item="item"/>
                    </div>
                  </div>
                </div>
              </div>
            </div>

            <div class="block-tags block">
              <b-taglist>
                <a v-for="(c, idx) in item.cast" :key="'cast' + idx" @click='showCastScenes([c.name])'
                   class="tag is-warning is-small">{{ c.name }} ({{ c.count }})</a>
                <a v-for="(tag, idx) in item.tags" :key="'tag' + idx" @click='showTagScenes([tag.name])'
                   class="tag is-info is-small">{{ tag.name }} ({{ tag.count }})</a>
              </b-taglist>
            </div>

            <div class="block-opts block">
              <b-tabs v-model="activeTab" :animated="false">

                <b-tab-item :label="`Files (${fileCount})`">
                  <div class="block-tab-content block">
                    <div class="content media is-small" v-for="(f, idx) in filesByType" :key="idx">
                      <div class="media-left">
                        <button rounded class="button is-success is-small" @click='playFile(f)'
                                v-show="f.type === 'video'">
                          <b-icon pack="fas" icon="play" size="is-small"></b-icon>
                        </button>
                        <b-tooltip :label="$t('Select this script for export')" position="is-right">
                        <button rounded class="button is-info is-small is-outlined" @click='selectScript(f)'
                          v-show="f.type === 'script'" v-bind:class="{ 'is-success': f.is_selected_script, 'is-info' :!f.is_selected_script }">
                          <b-icon pack="mdi" icon="pulse"></b-icon>
                        </button>
                        </b-tooltip>
                      </div>
                      <div class="media-content" style="overflow-wrap: break-word;">
                        <strong>{{ f.filename }}</strong><br/>
                        <small>
                          <span class="pathDetails">{{ f.path }}</span>
                          <br/>
                          {{ prettyBytes(f.size) }},
                          <span v-if="f.type === 'video'">{{ f.video_width }}x{{ f.video_height }},</span>
                          {{ format(parseISO(f.created_time), "yyyy-MM-dd") }}
                        </small>
                        <div v-if="f.type === 'script' && f.has_heatmap" class="heatmapFunscript">
                          <img :src="getHeatmapURL(f.id)"/>
                        </div>
                      </div>
                      <div class="media-right">
                        <button class="button is-danger is-small is-outlined" @click='removeFile(f)'>
                          <b-icon pack="fas" icon="trash" size="is-small"></b-icon>
                        </button>
                      </div>
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
                    <div class="content cuepoint-list">
                      <ul>
                        <li v-for="(c, idx) in sortedCuepoints" :key="idx">
                          <code>{{ humanizeSeconds(c.time_start) }}</code> -
                          <a @click="playCuepoint(c)"><strong>{{ c.name }}</strong></a>
                          <button class="button is-danger is-outlined is-small" @click="deleteCuepoint(c)" title="Delete cuepoint">
                            <b-icon pack="fas" icon="trash" />
                          </button>
                        </li>
                      </ul>
                    </div>
                  </div>
                </b-tab-item>

                <b-tab-item label="Watch history">
                  <div class="block-tab-content block">
                    <div>
                      {{ historySessionsCount }} view sessions, total duration
                      {{ humanizeSeconds(historySessionsDuration) }}
                    </div>
                    <div class="content is-small">
                      <div class="block" v-for="(session, idx) in item.history" :key="idx">
                        <strong>{{ format(parseISO(session.time_start), "yyyy-MM-dd kk:mm:ss") }} -
                          {{ humanizeSeconds(session.duration) }}</strong>
                      </div>
                    </div>
                  </div>
                </b-tab-item>

                <b-tab-item label="Description">
                  <div class="block-tab-content block">
                    <b-message>
                      {{ item.synopsis }}
                    </b-message>
                  </div>
                </b-tab-item>

              </b-tabs>
            </div>

          </div>
        </div>
      </section>
    </div>
    <button class="modal-close is-large" aria-label="close" @click="close()"></button>
    <a class="prev" @click="prevScene" v-if="$store.getters['sceneList/prevScene'](item) !== null"
       title="Keyboard shortcut: O">&#10094;</a>
    <a class="next" @click="nextScene" v-if="$store.getters['sceneList/nextScene'](item) !== null"
       title="Keyboard shortcut: P">&#10095;</a>
  </div>
</template>

<script>
import ky from 'ky'
import videojs from 'video.js'
import 'videojs-vr/dist/videojs-vr.min.js'
import { format, formatDistance, parseISO } from 'date-fns'
import prettyBytes from 'pretty-bytes'
import VueLoadImage from 'vue-load-image'
import GlobalEvents from 'vue-global-events'
import StarRating from 'vue-star-rating'
import FavouriteButton from '../../components/FavouriteButton'
import WatchlistButton from '../../components/WatchlistButton'
import EditButton from '../../components/EditButton'

export default {
  name: 'Details',
  components: { VueLoadImage, GlobalEvents, StarRating, WatchlistButton, FavouriteButton, EditButton },
  data () {
    return {
      index: 1,
      activeTab: 0,
      activeMedia: 0,
      player: {},
      tagAct: '',
      tagPosition: '',
      cuepointPositionTags: ['', 'standing', 'sitting', 'laying', 'kneeling'],
      cuepointActTags: ['', 'handjob', 'blowjob', 'doggy', 'cowgirl', 'revcowgirl', 'missionary', 'titfuck', 'anal', 'cumshot', '69', 'facesit'],
      carouselSlide: 0
    }
  },
  computed: {
    item () {
      const item = this.$store.state.overlay.details.scene
      if (this.$store.state.optionsWeb.web.tagSort === 'alphabetically') {
        item.tags.sort((a, b) => a.name < b.name ? -1 : 1)
      }
      return item
    },
    // Properties for gallery
    images () {
      return JSON.parse(this.item.images)
    },
    // Tab: cuepoints
    sortedCuepoints () {
      if (this.item.cuepoints !== null) {
        return this.item.cuepoints.slice().sort((a, b) => (a.time_start > b.time_start) ? 1 : -1)
      }
      return []
    },
    // Tab: files
    fileCount () {
      if (this.item.file !== null) {
        return this.item.file.length
      }
      return 0
    },
    filesByType () {
      if (this.item.file !== null) {
        return this.item.file.slice().sort((a, b) => (a.type === 'video') ? -1 : 1)
      }
      return []
    },
    // Tab: history
    historySessionsCount () {
      if (this.item.history !== null) {
        return this.item.history.length
      }
      return 0
    },
    historySessionsDuration () {
      if (this.item.history !== null) {
        let total = 0
        this.item.history.slice().map(i => {
          total = total + i.duration
          return 0
        })
        return total
      }
      return 0
    },
    showEdit () {
      return this.$store.state.overlay.edit.show
    }
  },
  mounted () {
    this.setupPlayer()
  },
  methods: {
    setupPlayer () {
      this.player = videojs(this.$refs.player, {
        aspectRatio: '1:1',
        fluid: true,
        loop: true
      })

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
              this.player.dispose()
              this.$store.commit('overlay/hideDetails')
            }
          }
        }
      })
    },
    updatePlayer (src, projection) {
      this.player.reset()

      /* const vr = */ this.player.vr({
        projection: projection,
        forceCardboard: false
      })

      this.player.on('loadedmetadata', function () {
        // vr.camera.position.set(-1, 0, 2);
      })

      if (src) {
        this.player.src({ src: src, type: 'video/mp4' })
      }
      this.player.poster(this.getImageURL(this.item.cover_url, ''))
    },
    showCastScenes (actor) {
      this.$store.state.sceneList.filters.cast = actor
      this.$store.state.sceneList.filters.sites = []
      this.$store.state.sceneList.filters.tags = []
      this.$router.push({
        name: 'scenes',
        query: { q: this.$store.getters['sceneList/filterQueryParams'] }
      })
      this.close()
    },
    showTagScenes (tag) {
      this.$store.state.sceneList.filters.cast = []
      this.$store.state.sceneList.filters.sites = []
      this.$store.state.sceneList.filters.tags = tag
      this.$router.push({
        name: 'scenes',
        query: { q: this.$store.getters['sceneList/filterQueryParams'] }
      })
      this.close()
    },
    playPreview () {
      this.activeMedia = 1
      this.updatePlayer('/api/dms/preview/' + this.item.scene_id, 'NONE')
      this.player.play()
    },
    playFile (file) {
      this.activeMedia = 1
      this.updatePlayer('/api/dms/file/' + file.id + '?dnt=true', '180')
      this.player.play()
    },
    removeFile (file) {
      this.$buefy.dialog.confirm({
        title: 'Remove file',
        message: `You're about to remove file <strong>${file.filename}</strong> from <strong>disk</strong>.`,
        type: 'is-danger',
        hasIcon: true,
        onConfirm: () => {
          ky.delete(`/api/files/file/${file.id}`).json().then(data => {
            this.$store.commit('overlay/showDetails', { scene: data })
          })
        }
      })
    },
    selectScript (file) {
      ky.post(`/api/scene/selectscript/${this.item.id}`, {
        json: {
          file_id: file.id,
        }
      }).json().then(data => {
          this.$store.commit('overlay/showDetails', { scene: data })
      })
    },
    getImageURL (u, size) {
      if (u.startsWith('http') || u.startsWith('https')) {
        return '/img/' + size + '/' + u.replace('://', ':/')
      } else {
        return u
      }
    },
    getIndicatorURL (idx) {
      if (this.images[idx] !== undefined) {
        return this.getImageURL(this.images[idx].url, 'x40')
      } else {
        return '/ui/images/blank.png'
      }
    },
    getHeatmapURL (fileId) {
      return `/api/dms/heatmap/${fileId}`
    },
    playCuepoint (cuepoint) {
      this.player.currentTime(cuepoint.time_start)
      this.player.play()
    },
    addCuepoint () {
      let name = ''
      if (this.tagAct !== '') {
        name = this.tagAct
      }
      if (this.tagPosition !== '') {
        name = this.tagPosition
      }
      if (this.tagPosition !== '' && this.tagAct !== '') {
        name = `${this.tagPosition}-${this.tagAct}`
      }
      ky.post(`/api/scene/${this.item.id}/cuepoint`, {
        json: {
          name: name,
          time_start: this.player.currentTime()
        }
      }).json().then(data => {
        this.$store.commit('overlay/showDetails', { scene: data })
      })
    },
    deleteCuepoint (cuepoint) {
      ky.delete(`/api/scene/${this.item.id}/cuepoint/${cuepoint.id}`)
        .json().then(data => {
          this.$store.commit('overlay/showDetails', { scene: data })
        })
    },
    close () {
      this.player.dispose()
      this.$store.commit('overlay/hideDetails')
    },
    humanizeSeconds (seconds) {
      return new Date(seconds * 1000).toISOString().substr(11, 8)
    },
    setRating (val) {
      ky.post(`/api/scene/rate/${this.item.id}`, { json: { rating: val } })

      const updatedScene = Object.assign({}, this.item)
      updatedScene.star_rating = val
      this.$store.commit('sceneList/updateScene', updatedScene)
    },
    nextScene () {
      const data = this.$store.getters['sceneList/nextScene'](this.item)
      if (data !== null) {
        this.$store.commit('overlay/showDetails', { scene: data })
        this.activeMedia = 0
        this.carouselSlide = 0
        this.updatePlayer(undefined, '180')
      }
    },
    prevScene () {
      const data = this.$store.getters['sceneList/prevScene'](this.item)
      if (data !== null) {
        this.$store.commit('overlay/showDetails', { scene: data })
        this.activeMedia = 0
        this.carouselSlide = 0
        this.updatePlayer(undefined, '180')
      }
    },
    playerStepBack () {
      const wasPlaying = !this.player.paused()
      if (wasPlaying) {
        this.player.pause()
      }
      let seekTime = this.player.currentTime() - 5
      if (seekTime <= 0) {
        seekTime = 0
      }
      this.player.currentTime(seekTime)
      if (wasPlaying) {
        this.player.play()
      }
    },
    playerStepForward () {
      const duration = this.player.duration()
      const wasPlaying = !this.player.paused()
      if (wasPlaying) {
        this.player.pause()
      }
      let seekTime = this.player.currentTime() + 5
      if (seekTime >= duration) {
        seekTime = wasPlaying ? duration - 0.001 : duration
      }
      this.player.currentTime(seekTime)
      if (wasPlaying) {
        this.player.play()
      }
    },
    toggleGallery () {
      this.activeMedia = 0
    },
    format,
    parseISO,
    prettyBytes,
    formatDistance
  }
}
</script>

<style lang="less" scoped>
.bbox {
  flex: 1 0 calc(25%);
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  padding: 0;
  line-height: 0;
}

.is-1by1 {
  padding-top: calc(100% - 40px - 1em) !important;
}

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

span.is-active img {
  border: 2px;
}

.pathDetails {
  color: #b0b0b0;
}

.cuepoint-list li > button {
  margin-left: 7px;
}

.heatmapFunscript {
  width: 100%;
  padding: 0;
  margin-top: 0.5em;
}

.heatmapFunscript img {
  border: 1px #888 solid;
  width: 100%;
  height: 20px;
  margin: 0;
  padding: 0;
}
</style>
