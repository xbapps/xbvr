<template>
  <div class="modal is-active">
    <GlobalEvents
      :filter="e => !['INPUT', 'TEXTAREA'].includes(e.target.tagName)"
      @keyup.esc="close"
      @keydown.left="handleLeftArrow"
      @keydown.right="handleRightArrow"
      @keydown.o="prevScene"
      @keydown.p="nextScene"
      @keydown.f="$store.commit('sceneList/toggleSceneList', {scene_id: item.scene_id, list: 'favourite'})"
      @keydown.exact.w="$store.commit('sceneList/toggleSceneList', {scene_id: item.scene_id, list: 'watchlist'})"
      @keydown.shift.w="$store.commit('sceneList/toggleSceneList', {scene_id: item.scene_id, list: 'watched'})"
      @keydown.t="$store.commit('sceneList/toggleSceneList', {scene_id: item.scene_id, list: 'trailerlist'})"
      @keydown.e="$store.commit('overlay/editDetails', {scene: item})"
      @keydown.g="toggleGallery"
      @keydown.48="setRating(0)"
    />

    <div class="modal-background"></div>

    <div class="modal-card">
      <section class="modal-card-body">
        <div class="columns">

          <div class="column is-half">
            <b-tabs v-model="activeMedia" position="is-centered" :animated="false">

              <b-tab-item label="Gallery">
                <b-carousel v-model="carouselSlide" @change="scrollToActiveIndicator" :autoplay="false" :indicator-inside="false">
                  <b-carousel-item v-for="(carousel, i) in images" :key="i">
                    <div class="image is-1by1 is-full"
                         v-bind:style="{backgroundImage: `url(${getImageURL(carousel.url, '700,fit')})`, backgroundSize: 'contain', backgroundPosition: 'center', backgroundRepeat: 'no-repeat'}"></div>
                  </b-carousel-item>
                  <template slot="indicators" slot-scope="props">
                      <span class="al image" style="width:max-content;">
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
                <b-field position="is-centered">
                  <b-field>
                    <b-tooltip v-for="(skipBack, i) in skipBackIntervals" class="is-size-7" :key="i" :active="skipBack == lastSkipBackInterval ? true : false" :label="$t('Keyboard shortcut: Left Arrow')"
                        position="is-top" type="is-primary is-light" >
                    <b-button class="tag is-small is-outlined is-info is-light"  @click="playerStepBack(skipBack)">
                      <b-icon v-if="skipBack == lastSkipBackInterval" pack="mdi" icon="arrow-left-thin" size="is-small"></b-icon> {{ skipBack }}</b-button>
                    </b-tooltip>
                  </b-field>
                  <b-field style="margin-left:1em">
                    <b-tooltip v-for="(skipForward, i) in skipForwardIntervals" :key="i" :active="skipForward == lastSkipFowardInterval ? true : false" :label="$t('Keyboard shortcut: Right Arrow')"
                        position="is-top" type="is-primary is-light" >
                    <b-button class="tag is-small is-outlined is-info is-light" @click="playerStepForward(skipForward)">
                      <b-icon v-if="skipForward == lastSkipFowardInterval" pack="mdi" icon="arrow-right-thin" size="is-small"></b-icon> +{{ skipForward }}</b-button>
                    </b-tooltip>
                  </b-field>
                </b-field>
             </b-tab-item>

            </b-tabs>

          </div>

          <div class="column is-half">

            <div class="block-info block">
              <div class="content">
                <h3>
                  <span v-if="item.title">{{ item.title }}</span>
                  <span v-else class="missing">(no title)</span>
                  <small class="is-pulled-right">
                    {{ format(parseISO(item.release_date), "yyyy-MM-dd") }}
                  </small>
                </h3>
                <div class="columns">
                  <div class="column pb-0">
                    <small>
                      <a :href="item.scene_url" target="_blank" rel="noreferrer">{{ item.site }}</a>
                      <br v-if="item.members_url != ''"/>
                      <a v-if="item.members_url != ''" :href="item.members_url" target="_blank" rel="noreferrer"><b-icon pack="mdi" icon="link-lock" custom-size="mdi-18px"/>Members Link</a>
                    </small>
                  </div>
                  <div class="column pb-0">
                    <small v-if="item.duration" class="is-pulled-right">{{ item.duration }} minutes</small>
                  </div>
                </div>
                <div class="columns is-vcentered">
                  <div class="column pt-0">
                    <b-field>
                      <star-rating :key="item.id" v-model="item.star_rating" :rating="item.star_rating" @rating-selected="setRating"
                                   :increment="0.5" :star-size="20" :show-rating="false" />
                      <b-tooltip :label="$t('Reset Rating')" position="is-right" :delay="250">
                        <b-icon pack="mdi" icon="autorenew" size="is-small" @click.native="setRating(0)" style="padding-left: 1em;padding-top: .5em;"/>
                      </b-tooltip>
                    </b-field>
                  </div>
                  <div class="column pt-0">
                    <div class="is-flex is-pulled-right" style="gap: 0.25rem">
                      <hidden-button :item="item"/>
                      <watchlist-button :item="item"/>
                      <trailerlist-button :item="item"/>
                      <favourite-button :item="item"/>
                      <wishlist-button :item="item"/>
                      <watched-button :item="item"/>
                      <edit-button :item="item"/>
                      <refresh-button :item="item"/>
                    </div>
                  </div>
                </div>
              </div>
            </div>

            <div class="image-row" v-if="activeTab != 1">
              <div v-for="(image, idx) in castimages" :key="idx" class="image-wrapper">
                <b-tooltip  type="is-light" :label="image.actor_label"  :delay=100>
                  <vue-load-image>
                    <img slot="image" :src="getImageURL(image.src)" alt="Image" class="thumbnail" @mouseover="showTooltip(idx)" @mouseout="hideTooltip(idx)" @click='showActorDetail([image.actor_id])' />
                    <img slot="preloader" :src="getImageURL('https://i.stack.imgur.com/kOnzy.gif')" style="height: 50px;display: block;margin-left:auto;margin-right: auto;" @click='showCastScenes([image.actor_name])' />
                    <img slot="error" src="/ui/images/blank_female_profile.png" width="80" @click='showActorDetail([image.actor_id])' />
                  </vue-load-image>
                </b-tooltip>

                <div v-if="image.visible" class="tooltip">
                  <img :src="getImageURL(image.src)" alt="Tooltip Image" />
                </div>
              </div>
            </div>

            <div class="block-tags block" v-if="activeTab != 1">
              <b-taglist>
                <a v-for="(c, idx) in item.cast" :key="'cast' + idx" @click='showCastScenes([c.name])'
                   class="tag is-warning is-small">{{ c.name }} ({{ c.avail_count }}/{{ c.count }})</a>
                <a @click='showSiteScenes([item.site])'
                   class="tag is-primary is-small">{{ item.site }}</a>
                <a v-for="(tag, idx) in item.tags" :key="'tag' + idx" @click='showTagScenes([tag.name])'
                   class="tag is-info is-small">{{ tag.name }} ({{ tag.count }})</a>
              </b-taglist>
            </div>

            <div class="block-tags block" v-if="activeTab == 1">
             <b-taglist>
              <b-tooltip  type="is-danger" :label="disableSaveMsg()" position="is-right" :delay=250 :active="disableSaveButtons()">
                <b-button @click="updateCuepoint(false)" class="tag is-info is-small is-warning" accesskey="a" :disabled="disableSaveButtons()" >
                  <u>A</u>dd New
                </b-button>
              </b-tooltip>
                <b-button @click="vidPosition = new Date(0,0,0,0,0, 0, player.currentTime() * 1000)" class="tag is-info is-small is-warning" accesskey="t">Current <u>T</u>ime</b-button>
              <b-tooltip type="is-danger" :label="$t(disableSaveMsg())" position="is-right" :delay=250 :active="disableSaveButtons()">
                <b-button v-if="currentCuepointId > 0" @click="updateCuepoint(true)" class="tag is-info is-small is-warning" accesskey="s"
                  :disabled="disableSaveButtons()" >
                  <u>S</u>ave Edit
                </b-button>
              </b-tooltip>
                <b-button v-if="cuepointName!=''" @click='cuepointName=""' class="tag is-info is-small is-warning" >Clear Cuepoint Name</b-button>
                <b-button v-if="tagAct!=''" @click='setCuepointName("")' class="tag is-info is-small is-warning" accesskey="c"><u>C</u>lear Action</b-button>
              </b-taglist>
            </div>

            <div class="is-divider" data-content="Cuepoint Positions" v-if="activeTab == 1"></div>
            <div class="block-tags block" v-if="activeTab == 1">
              <b-taglist>
                <b-button v-for="(c, idx) in cuepointPositionTags.slice(1)" :key="'pos' + idx" @click='setCuepointName([c])' class="tag is-info is-small">{{c}}</b-button>
              </b-taglist>
            </div>
            <div class="is-divider" data-content="Default Cuepoint Actions" v-if="activeTab == 1"></div>
            <div class="block-tags block" v-if="activeTab == 1">
              <b-taglist>
                <b-button v-for="(c, idx) in cuepointActTags.slice(1)" :key="'action' + idx" @click='setCuepointName([c])' class="tag is-info is-small">{{c}}</b-button>
              </b-taglist>
            </div>
            <div class="is-divider" data-content="Cast Cuepoints" v-if="activeTab == 1"></div>
            <div class="block-tags block" v-if="activeTab == 1">
              <b-taglist>
                <b-button v-for="(c, idx) in item.cast" :key="'cast' + idx" @click='setCuepointName([c.name])' class="tag is-info is-small">{{c.name}}</b-button>
              </b-taglist>
            </div>
            <div class="is-divider" data-content="Scene Tag Cuepoints" v-if="activeTab == 1"></div>
            <div class="block-tags block" v-if="activeTab == 1">
              <b-taglist>
                <b-button v-for="(tag, idx) in item.tags" :key="'tag' + idx" @click='setCuepointName([tag.name])'
                   class="tag is-info is-small">{{ tag.name }}</b-button>
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
                        <button rounded class="button is-info is-small is-outlined" disabled
                                v-show="f.type === 'hsp'">
                          <b-icon pack="mdi" icon="safety-goggles"></b-icon>
                        </button>
                        <button rounded class="button is-info is-small is-outlined" disabled
                                v-show="f.type === 'subtitles'">
                          <b-icon pack="mdi" icon="subtitles"></b-icon>
                        </button>
                      </div>
                      <div class="media-content" style="overflow-wrap: break-word;">
                        <strong>{{ f.filename }}</strong><br/>
                        <small>
                          <span class="pathDetails">{{ f.path }}</span>
                          <br/>
                          {{ prettyBytes(f.size) }}<span v-if="f.type === 'video'"> ({{ prettyBytes(f.video_bitrate, { bits: true })  }}/s)</span>,
                          <span v-if="f.type === 'video'"><span class="videosize">{{ f.video_width }}x{{ f.video_height }} {{ f.video_codec_name }}</span>, {{ f.projection }},&nbsp;</span>
                          <span v-if="f.duration > 1">{{ humanizeSeconds(f.duration) }},</span>
                          {{ format(parseISO(f.created_time), "yyyy-MM-dd") }}
                        </small>
                        <div v-if="f.type === 'script' && f.has_heatmap" class="heatmapFunscript">
                          <img :src="getHeatmapURL(f.id)"/>
                        </div>
                      </div>
                      <div class="media-right">
                        <button class="button is-dark is-small is-outlined" title="Unmatch file from scene" @click='unmatchFile(f)'>
                          <b-icon pack="fas" icon="unlink" size="is-small"></b-icon>
                        </button>&nbsp;
                        <button class="button is-danger is-small is-outlined" title="Delete file from disk" @click='removeFile(f)'>
                          <b-icon pack="fas" icon="trash" size="is-small"></b-icon>
                        </button>
                      </div>
                    </div>
                  </div>
                </b-tab-item>

                <b-tab-item :label="`Cuepoints (${sortedCuepoints.length})`">
                  <div class="block-tab-content block">
                    <div class="block" >
                      <div class="columns">
                        <div class="column is-2">
                        <b-field label="Track" width="7.25em" label-position="on-border">
                          <b-input v-model="track" width="7.25em"></b-input>
                        </b-field>
                        </div>
                        <div class="column">
                        <b-field label="Name" label-position="on-border">
                          <b-autocomplete v-model="cuepointName" :data="filteredCuepointPositionList" :open-on-focus="true"></b-autocomplete>
                        </b-field>
                        </div>
                        <div class="column is-2">
                        <b-field label="Start" label-position="on-border">
                          <b-timepicker v-model="vidPosition" rounded editable placeholder="Defaults to player position" hour-format="24" :enable-seconds="true" :max-time="maxTime" :time-formatter="timeFormatter" :time-parser="timeParser" >
                          <b-button
                            label="Current Time"
                            type="is-primary"
                            @click="vidPosition = new Date(0,0,0,0,0, 0, player.currentTime() * 1000)" />
                          </b-timepicker>
                        </b-field>
                        </div>
                        <div class="column is-2">
                          <b-field label="End" label-position="on-border">
                          <b-timepicker v-model="endTime" rounded editable placeholder="Defaults to player position" hour-format="24" :enable-seconds="true" :max-time="maxTime" :time-formatter="timeFormatter" :time-parser="timeParser" >
                          <b-button
                            label="Current Time"
                            type="is-primary"
                            @click="endTime = new Date(0,0,0,0,0, 0, player.currentTime() * 1000)" />
                          </b-timepicker>
                        </b-field>
                        </div>
                      </div>
                    </div>
                    <div>
                      <!-- :sort-multiple="sortMultiple" :sort-multiple-data="cuepointSorting" -->
                        <b-table :data="sortedCuepoints"  :narrowed=true :per-page=7 focusable striped sticky-header
                          @select="cuepointSelected">
                          <!-- paginated  pagination-position="top" :pagination-rounded=true pagination-size="is-small" -->
                          <b-table-column field="track" label="Track" width="7.25em" v-slot="props" >
                            {{ props.row.track ==null ? "" :  props.row.track }}
                          </b-table-column>
                          <b-table-column field="name" label="Name" v-slot="props"  is-small>
                            {{ props.row.name }}
                          </b-table-column>
                          <b-table-column field="time_start" label="Start" v-slot="props" width="6.5em"  >
                            {{ humanizeSeconds1DP(props.row.time_start) }}
                          </b-table-column>
                          <b-table-column field="time_end" label="End" v-slot="props" width="6.5em"  >
                            {{ props.row.time_end==null ? "" :  humanizeSeconds1DP(props.row.time_end) }}
                          </b-table-column>
                          <b-table-column field="rating" v-slot="props" width="7em"  >
                            <b-field v-if="props.row.track!=null">
                              <star-rating :key="props.row.id" v-model="props.row.rating" :rating="props.row.rating" @rating-selected="setCuepointRating(props.row)" :increment="0.5" :star-size="10" />
                              <b-icon v-if="props.row.rating>0" pack="mdi" icon="autorenew" size="is-small" @click.native="clearCuepointRating(props.row)" style="padding-left: .25em;padding-top: .5em;"/>
                            </b-field>
                          </b-table-column>
                          <b-table-column v-slot="props" width="1em" >
                            <button class="button is-danger is-outlined is-small" @click="deleteCuepoint(props.row.id)" title="Delete cuepoint">
                              <b-icon pack="fas" icon="trash" />
                            </button>
                          </b-table-column>
                        </b-table>
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
                <b-tab-item v-if="this.$store.state.optionsAdvanced.advanced.showSceneSearchField" label="Search fields">
                  <div class="block-tab-content block">
                    <div class="content is-small">
                      <div class="block" v-for="(field, idx) in searchfields" :key="idx">
                        <strong>{{ field.fieldName }} - </strong> {{ field.fieldValue }}
                      </div>
                    </div>
                  </div>
                </b-tab-item>

              </b-tabs>
            </div>

          </div>
        </div>
      </section>
      <div class="scene-id">
        {{ item.scene_id }}
        <span  v-if="this.$store.state.optionsAdvanced.advanced.showInternalSceneId">{{ $t('Internal ID') }}: {{item.id}}</span>
        <a v-if="this.$store.state.optionsAdvanced.advanced.showHSPApiLink" :href="`/heresphere/${item.id}`" target="_blank" rel="noreferrer" style="margin-left:0.5em">
          <img src="/ui/icons/heresphere_24.png" style="height:15px;"/>
        </a>
      </div>
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
import WishlistButton from '../../components/WishlistButton'
import WatchedButton from '../../components/WatchedButton'
import EditButton from '../../components/EditButton'
import RefreshButton from '../../components/RefreshButton'
import TrailerlistButton from '../../components/TrailerlistButton'
import HiddenButton from '../../components/HiddenButton'

export default {
  name: 'Details',
  components: { VueLoadImage, GlobalEvents, StarRating, WatchlistButton, FavouriteButton, WishlistButton, WatchedButton, EditButton, RefreshButton, TrailerlistButton, HiddenButton },
  data () {
    return {
      index: 1,
      activeTab: 0,
      activeMedia: 0,
      player: {},
      tagAct: '',
      cuepointName: '',
      cuepointRating: 0,
      cuepointPositionTags: ['', 'standing', 'sitting', 'laying', 'kneeling'],
      cuepointActTags: ['', 'handjob', 'blowjob', 'doggy', 'cowgirl', 'revcowgirl', 'missionary', 'titfuck', 'anal', 'cumshot', '69', 'facesit'],
      carouselSlide: 0,
      vidPosition: null,
      skipForwardIntervals: [5, 10, 30, 60, 120, 300],
      skipBackIntervals: [-300, -120, -60, -30, -10, -5],
      lastSkipFowardInterval: 5,
      lastSkipBackInterval: -5,
      currentCuepointId: 0,
      maxTime: new Date(0, 0, 0, 5, 0, 0),
      cuepointSorting: [{ field: "is_hsp", order: "asc" },{ field: "time_start", order: "desc" }, {field: "track", order: "desc"}, {field: "time_end", order: "desc"}],
      trackInput: '',
      track: null,
      endTime: null,
      sortMultiple: true,
      castimages: [],
      searchfields: [],
    }
  },
  computed: {
    item () {
      const item = this.$store.state.overlay.details.scene
      if (this.$store.state.optionsWeb.web.tagSort === 'alphabetically') {
        item.tags.sort((a, b) => a.name < b.name ? -1 : 1)
      }
      let releasedate = parseISO(item.release_date)
      let imgs = item.cast.map((actor) => {
        let birthdate = parseISO(actor.birth_date)
        let label = actor.name
        if (birthdate.getFullYear() > 0) {
          let age = releasedate.getFullYear() - birthdate.getFullYear()
          if ((releasedate.getMonth() < birthdate.getMonth()) || (releasedate.getMonth() == birthdate.getMonth() && releasedate.getDate() < birthdate.getDate())) {
            age -= 1
          }
          label += `, ${age} in scene`
        }
        let img = actor.image_url
        if (img == "" ){
          img = "blank"  // forces an error image to load, blank won't display an image
        }
        if (actor.name.startsWith("aka:")) {
          img = ""
        }
        return {src: img, visible: false, actor_name: actor.name, actor_label: label, actor_id: actor.id};
      });

      this.castimages =  imgs.filter((img) => {
        return img.src !== '';
        });
      return item
    },
    // Properties for gallery
    images () {
      if (this.item.images=="null") {
        return "[]"
      }
      return JSON.parse(this.item.images).filter(im => im && im.url)
    },
    // Tab: cuepoints
    sortedCuepoints () {
      if (this.item.cuepoints !== null) {
        for (let i = 0; i < this.item.cuepoints.length; i++) {
          this.item.cuepoints[i].is_hsp = this.item.cuepoints[i].track == null ? 0 : 1
        }
        let x=this.item.cuepoints.slice().sort((a, b) => (a.time_start > b.time_start) ? 1 : -1 || (a.is_hsp >b.is_hsp) ? 1 : -1 )
        x=this.item.cuepoints.slice().sort((a,b) => {
          let compare = (a.is_hsp<b.is_hsp) ? -1 : (a.is_hsp>b.is_hsp) ? 1 : 0
          if (compare!=0) {
            return compare
          }
          compare = (a.time_start<b.time_start) ? -1 : (a.time_start>b.time_start) ? 1 : 0
          if (compare!=0) {
            return compare
          }
          compare = (a.track<b.track) ? -1 : (a.track>b.track) ? 1 : 0
          if (compare!=0) {
            return compare
          }
          return  (a.time_end<b.time_end) ? -1 : (a.time_end>b.time_end) ? 1 : 0
        })
        return x
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
    },
    filteredCuepointPositionList () {
      // filter the list of positions based on what has been entered so far
      let list=this.cuepointActTags.concat(this.cuepointPositionTags)
      return list.filter((option) => {
        return option
          .toString()
          .toLowerCase()
          .trim()
          .indexOf(this.cuepointName.toString().toLowerCase()) >= 0
      })
    },
  },
  mounted () {
    this.setupPlayer()

    // load default cuepoint actions & positions from kv entry in the db
    ky.get('/api/options/cuepoints').json().then(data => {
      this.cuepointActTags = data.actions
      this.cuepointPositionTags = data.positions
      this.cuepointActTags.unshift("")
      this.cuepointPositionTags.unshift("")
      })
    this.getSearchFields()
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
        enableVolumeScroll: false,
        customKeys: {
          closeModal: {
            key: function (event) {
              return event.which === 27
            },
            handler: (player, options, event) => {
              this.player.dispose()
              this.$store.commit('overlay/hideDetails')
            }
          },
          zoomIn: {
            handler: (player, options, event) => {
              this.zoomHandler(true)
            }
          },
          zoomOut: {
            handler: (player, options, event) => {
              this.zoomHandler(false)
            }
          }
        }
      })

      const videoElement = this.player.el();
      videoElement.addEventListener('wheel', this.zoomHandlerWeb.bind(this))
    },

    zoomHandlerWeb(event) {
      event.preventDefault();
      this.zoomHandler(event.deltaY < 0)
    },

    zoomHandler(isZoomingIn) {
      const vr = this.player.vr()
      const minFov = 30
      const maxFov = 130
      let fov = vr.camera.fov + (isZoomingIn ? -1 : 1)

      if (fov < minFov) {
        fov = minFov
      }

      if (fov > maxFov) {
        fov = maxFov
      }

      vr.camera.fov = fov;
      vr.camera.updateProjectionMatrix()
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
      this.$store.state.sceneList.filters.attributes = []
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
      this.$store.state.sceneList.filters.attributes = []
      this.$router.push({
        name: 'scenes',
        query: { q: this.$store.getters['sceneList/filterQueryParams'] }
      })
      this.close()
    },
    showSiteScenes (site) {
      this.$store.state.sceneList.filters.cast = []
      this.$store.state.sceneList.filters.sites = site
      this.$store.state.sceneList.filters.tags = []
      this.$store.state.sceneList.filters.attributes = []
      this.$router.push({
        name: 'scenes',
        query: { q: this.$store.getters['sceneList/filterQueryParams'] }
      })
      this.close()
    },
    showActorDetail (actor_id) {
      ky.get('/api/actor/'+actor_id).json().then(data => {
        if (data.id != 0){
          this.$store.commit('overlay/showActorDetails', { actor: data })
          this.close()
        }
      })
    },
    playPreview () {
      this.activeMedia = 1
      this.updatePlayer('/api/dms/preview/' + this.item.scene_id, 'NONE')
      this.player.play()
    },
    playFile (file) {
      this.activeMedia = 1
      this.updatePlayer('/api/dms/file/' + file.id + '?dnt=true', (file.projection == 'flat' ? 'NONE' : '180'))
      this.player.play()
    },
    unmatchFile (file) {
      this.$buefy.dialog.confirm({
        title: 'Unmatch file',
        message: `You're about to unmatch the file <strong>${file.filename}</strong> from this scene. Afterwards, it can be matched again to this or any other scene.`,
        type: 'is-info is-wide',
        hasIcon: true,
        id: 'heh',
        onConfirm: () => {
          ky.post(`/api/files/unmatch`, {json:{file_id: file.id}}).json().then(data => {
            this.$store.commit('overlay/showDetails', { scene: data })
          })
        }
      })
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
      if (u==undefined) {
        return u
      }
      try {
        if (u.startsWith('http') || u.startsWith('https')) {
          return '/img/' + size + '/' + encodeURI(u)
        } else {
          return u
        }
      } catch {
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
      // populate the cuepoint edit fields
      this.vidPosition = new Date(0, 0, 0, 0, 0, 0, cuepoint.time_start*1000)
      this.endTime = new Date(0, 0, 0, 0, 0, 0, cuepoint.time_end*1000)
      this.currentCuepointId = cuepoint.id
      this.cuepointRating = cuepoint.rating
      if (cuepoint.name.indexOf('-') > 0) {
        this.cuepointName = cuepoint.name.substr(0, cuepoint.name.indexOf('-'))
        this.tagAct = cuepoint.name.substr(cuepoint.name.indexOf('-') + 1)
      } else {
        this.tagAct = cuepoint.name
        this.cuepointName = ''
      }
      // now mow the player position
      this.player.currentTime(cuepoint.time_start)
      this.player.play()
    },
    updateCuepoint (editCuepoint) {
      if (this.disableSaveButtons()) return
      // if edit choosen, delete existing cuepoint before add
      if (editCuepoint && this.currentCuepointId > 0) {
        this.deleteCuepoint(this.currentCuepointId)
      }
      let name =  this.cuepointName
      let pos = this.player.currentTime()
      let endpos=null
      this.track=parseInt(this.track)
      if (this.vidPosition != null) {
        pos = (this.vidPosition.getMilliseconds() / 1000) + this.vidPosition.getSeconds() + (this.vidPosition.getMinutes() * 60) + (this.vidPosition.getHours() * 60 * 60)
      }
      if (this.endTime != null) {
        endpos = (this.endTime.getMilliseconds() / 1000) + this.endTime.getSeconds() + (this.endTime.getMinutes() * 60) + (this.endTime.getHours() * 60 * 60)
      }
      this.currentCuepointId = 0

      ky.post(`/api/scene/${this.item.id}/cuepoint`, {
        json: {
          track: this.track,
          name: name,
          time_start: pos,
          time_end: endpos,
          rating: this.cuepointRating
        }
      }).json().then(data => {
        this.vidPosition = null
        this.endTime = null
        this.cuepointName=''
        this.track = null
        this.$store.commit('sceneList/updateScene', data)
        this.$store.commit('overlay/showDetails', { scene: data })
      })
    },
    deleteCuepoint (cuepointid) {
      ky.delete(`/api/scene/${this.item.id}/cuepoint/${cuepointid}`)
        .json().then(data => {
          this.$store.commit('sceneList/updateScene', data)
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
    humanizeSeconds1DP (seconds) {
      return new Date(seconds * 1000).toISOString().substr(11, 10)
    },
    setRating (val) {
      ky.post(`/api/scene/rate/${this.item.id}`, { json: { rating: val } })

      const updatedScene = Object.assign({}, this.item)
      updatedScene.star_rating = val
      this.item.star_rating = val
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
      this.getSearchFields()
    },
    prevScene () {
      const data = this.$store.getters['sceneList/prevScene'](this.item)
      if (data !== null) {
        this.$store.commit('overlay/showDetails', { scene: data })
        this.activeMedia = 0
        this.carouselSlide = 0
        this.updatePlayer(undefined, '180')
      }
      this.getSearchFields()
    },
    playerStepBack (interval) {
      const wasPlaying = !this.player.paused()
      if (wasPlaying) {
        this.player.pause()
      }
      let seekTime = this.player.currentTime() + interval
      if (seekTime <= 0) {
        seekTime = 0
      }
      this.player.currentTime(seekTime)
      if (wasPlaying) {
        this.player.play()
      }
      this.lastSkipBackInterval = interval
    },
    playerStepForward (interval) {
      const duration = this.player.duration()
      const wasPlaying = !this.player.paused()
      if (wasPlaying) {
        this.player.pause()
      }
      let seekTime = this.player.currentTime() + interval
      if (seekTime >= duration) {
        seekTime = wasPlaying ? duration - 0.001 : duration
      }
      this.player.currentTime(seekTime)
      if (wasPlaying) {
        this.player.play()
      }
      this.lastSkipFowardInterval = interval
    },
    setCuepointName (param) {
      if (this.activeTab === 1) {
        if (this.cuepointName=='') {
          this.cuepointName = param.toString()
        }else{
          this.cuepointName = this.cuepointName+'-'+param.toString()
        }
      }
    },
    toggleGallery () {
      if (this.activeMedia == 0) {
        this.activeMedia = 1
      } else {
        this.activeMedia = 0
        }
    },
    handleLeftArrow () {
      if (this.activeMedia === 0)
      {
        this.carouselSlide = this.carouselSlide - 1
      } else {
        this.playerStepBack(this.lastSkipBackInterval)
      }
    },
    handleRightArrow () {
      if (this.activeMedia === 0)
      {
        this.carouselSlide = this.carouselSlide + 1
      } else {
        this.playerStepForward(this.lastSkipFowardInterval)
      }
    },
    scrollToActiveIndicator (value) {
      const indicators = document.querySelector('.carousel-indicator')
      const active = indicators.children[value]
      indicators.scrollTo({
        top: 0,
        left: active.offsetLeft + active.offsetWidth / 2 - indicators.offsetWidth / 2,
        behavior: 'smooth'
      })
    },
    timeFormatter(time) {
       return new Intl.DateTimeFormat('en', { hourCycle: 'h23', hour: "2-digit", minute: "2-digit", second: "2-digit", fractionalSecondDigits: 1 }).format(time)
    },
    timeParser(inputString) {
      let items = inputString.split(":")
      return new Date(0, 0, 0, items[0],items[1], 0, items[2]*1000)
    },
    cuepointSelected(cuepoint) {
      // populate the cuepoint edit fields
      this.vidPosition = new Date(0, 0, 0, 0, 0, 0, cuepoint.time_start*1000)
      this.endTime = new Date(0, 0, 0, 0, 0, 0, cuepoint.time_end*1000)
      this.currentCuepointId = cuepoint.id
      this.cuepointName = cuepoint.name
      this.track=cuepoint.track
      this.cuepointRating=cuepoint.rating
      // now mow the player position
      this.player.currentTime(cuepoint.time_start)
      this.player.play()
    },
    disableSaveButtons() {
      if (this.track!=null && this.track!="" && (isNaN(this.endTime) || this.endTime==null)) return true
      if ((this.track==null || this.track==="") && !isNaN(this.endTime) && this.endTime!=null) return true
      return false
    },
    disableSaveMsg() {
      if (this.track!=null && this.track!="" && (isNaN(this.endTime) || this.endTime==null)) return "Specify a End Time"
      if ((this.track==null || this.track==="") && !isNaN(this.endTime) && this.endTime!=null) return "End Time is only valid for HSP Cuepoints"
      return ""
    },
    setCuepointRating (row) {
      this.cuepointSelected(row)
      this.updateCuepoint(true)
    },
    clearCuepointRating (row) {
      row.rating=0
      this.cuepointSelected(row)
      this.updateCuepoint(true)
    },
    showTooltip(idx) {
      this.castimages[idx].visible = true;
    },
    hideTooltip(idx) {
      this.castimages[idx].visible = false;
    },
    getSearchFields() {
      // load search fields
      if (this.$store.state.optionsAdvanced.advanced.showSceneSearchField) {
        ky.get('/api/scene/searchfields', {
          searchParams: {
            q: this.item.id
          },
          }).json().then(data => {
            this.searchfields = data
          })
      }
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

.vue-star-rating {
    line-height: 0;
}

.scene-id {
  position: absolute;
  right:10px;
  bottom: 5px;
  font-size: 11px;
  color: #b0b0b0;
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
.videosize {
  color: rgb(60, 60, 60);
  font-weight: 550;
}

:deep(.carousel .carousel-indicator) {
  justify-content: flex-start;
  width: 100%;
  max-width: min-content;
  margin-left: auto;
  margin-right: auto;
  overflow: auto;
}
:deep(.carousel .carousel-indicator .indicator-item:not(.is-active)) {
  opacity: 0.5;
}
.is-divider {
  margin: .8rem 0;
}
.image-row {
  display: flex;
}
.image-wrapper {
  position: relative;
}
.thumbnail {
  height: 100px;
  margin-right: .5em;
  object-fit: cover;
}
.tooltip {
  position: absolute;
  z-index: 1;
  top: 50px;
  right: 100%;
  width: 400px;
  height: 400px;
  background-color: white;
  box-shadow: 0 2px 5px rgba(0, 0, 0, 0.3);
  display: flex;
  justify-content: center;
  align-items: center;
  padding: 10px;
  transform: translateX(10px);
}
.tooltip img {
  max-width: 100%;
  max-height: 100%;
}</style>
