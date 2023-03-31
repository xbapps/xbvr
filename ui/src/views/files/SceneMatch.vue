<template>
  <div class="modal is-active">
    <GlobalEvents
      :filter="e => !['INPUT', 'TEXTAREA'].includes(e.target.tagName)"
      @keyup.esc="close"
      @keydown.left="handleLeftArrow"
      @keydown.right="handleRightArrow"
      @keydown.o="prevFile"
      @keydown.p="nextFile"
    />
    <div class="modal-background"></div>
    <div class="modal-card">
      <header class="modal-card-head">
        <p class="modal-card-title">{{ $t("Match file to scene") }}</p>
        <button class="delete" @click="close" aria-label="close"></button>
      </header>
      <section class="modal-card-body">
        <div>
          <h6 class="title is-6">{{ file.filename }}</h6>
          <small>
            <span class="pathDetails">{{ file.path }}</span>
            <br/>
            {{ prettyBytes(file.size) }},
            <span v-if="file.type == 'video'">{{ file.video_width }}x{{ file.video_height }}, </span>
            <span v-if="file.duration > 0">{{ Math.floor(file.duration / 60) }} min,</span>
            {{ format(parseISO(file.created_time), "yyyy-MM-dd") }}
          </small>

          <b-field grouped>
            <b-tooltip :label="$t('Optional: select one or more words to target searching to a specific field')" :delay="500" position="is-right">
              <b-taglist>
                <b-tag class="tag is-info is-small">{{$t('Search Fields')}}</b-tag>
                <b-button @click='searchPrefix("title:")' class="tag is-info is-small is-light">title:</b-button>
                <b-button @click='searchPrefix("cast:")' class="tag is-info is-small is-light">cast:</b-button>
                <b-button @click='searchPrefix("site:")' class="tag is-info is-small is-light">site:</b-button>
                <b-button @click='searchPrefix("id:")' class="tag is-info is-small is-light">id:</b-button>
                <b-tooltip :label="$t('Defaults date range to the last week. Note:must match yyyy-mm-dd, include leading zeros')" :delay="500" position="is-top">
                  <b-button @click='searchDatePrefix("released:")' class="tag is-info is-small is-light">released:</b-button>
                  <b-button @click='searchDatePrefix("added:")' class="tag is-info is-small is-light">added:</b-button>
                </b-tooltip>
              </b-taglist>          
            </b-tooltip>
          </b-field>
          <b-field :label="$t('Search')" label-position="on-border">
            <div class="control">
              <input class="input" type="text" v-model='queryString' v-debounce:200ms="loadData" autofocus ref="searchInput">
            </div>
          </b-field>
          
          <b-table :data="data" ref="table" paginated :current-page.sync="currentPage" per-page="5">
            <b-table-column field="cover_url" :label="$t('Image')" width="120" v-slot="props">
              <vue-load-image>
                <img slot="image" :src="getImageURL(props.row.cover_url)"/>
                <img slot="preloader" src="/ui/images/blank.png"/>
                <img slot="error" src="/ui/images/blank.png"/>
              </vue-load-image>
            </b-table-column>
            <b-table-column field="site" :label="$t('Site')" sortable v-slot="props">
              <a :href="props.row.scene_url" target="_blank" rel="noreferrer">{{ props.row.site }}</a><br>
              <b-tooltip v-if="props.row.is_hidden" label="Flagged as Hidden"  :delay="250" >
                <b-tag type="is-info is-light" >
                  <b-icon pack="mdi" icon="eye-off-outline" size="is-small" style="margin-right:0.1em"/>                
                </b-tag>&nbsp;
              </b-tooltip>
              <b-tag type="is-info is-light" v-if="props.row.duration">
                <b-icon pack="mdi" icon="clock" size="is-small" style="margin-right:0.1em"/>
                {{props.row.duration}}
              </b-tag>&nbsp;
              <b-tag type="is-info is-light" v-if="videoFilesCount(props.row)">
                <b-icon pack="mdi" icon="file" size="is-small" style="margin-right:0.1em"/>
                {{videoFilesCount(props.row)}}
              </b-tag>&nbsp;
              <b-tag type="is-info is-light" v-if="props.row.is_scripted">
                <b-icon pack="mdi" icon="pulse" size="is-small"/>
                <span v-if="scriptFilesCount(props.row) > 1">{{scriptFilesCount(props.row)}}</span>
              </b-tag>&nbsp;
              <b-tag type="is-info is-light" v-if="subtitlesFilesCount(props.row)">
                <b-icon pack="mdi" icon="subtitles" size="is-small" style="margin-right:0.1em"/>
                {{subtitlesFilesCount(props.row)}}
              </b-tag>
            </b-table-column>
            <b-table-column field="title" :label="$t('Title')" sortable v-slot="props">
              <p v-if="props.row.title">{{ props.row.title }}</p>
              <small>
                <b-tag rounded v-for="i in props.row.cast" :key="i.id">{{ i.name }}</b-tag>
              </small>
            </b-table-column>
            <b-table-column field="release_date" :label="$t('Release date')" sortable nowrap v-slot="props">
              {{ format(parseISO(props.row.release_date), "yyyy-MM-dd") }}
            </b-table-column>
            <b-table-column field="scene_id" :label="$t('ID')" sortable nowrap v-slot="props">
              {{ props.row.scene_id }}
            </b-table-column>
            <b-table-column field="_score" :label="$t('Score')" sortable v-slot="props">
              <b-progress show-value :value="props.row._score * 100"></b-progress>
            </b-table-column>
            <b-table-column field="_assign" v-slot="props">
              <button class="button is-primary is-outlined" @click="assign(props.row.scene_id)">{{ $t("Assign") }}</button>
            </b-table-column>
          </b-table>
        </div>
      </section>
    </div>
    <a class="prev" @click="prevFile" title="Keyboard shortcut: O">&#10094;</a>
    <a class="next" @click="nextFile" title="Keyboard shortcut: P">&#10095;</a>
  </div>
</template>

<script>
import ky from 'ky'
import { format, parseISO } from 'date-fns'
import prettyBytes from 'pretty-bytes'
import VueLoadImage from 'vue-load-image'
import GlobalEvents from 'vue-global-events'

export default {
  name: 'SceneMatch',
  components: { VueLoadImage, GlobalEvents },
  data () {
    return {
      data: [],
      dataNumRequests: 0,
      dataNumResponses: 0,
      currentPage: 1,
      queryString: '',
      format,
      parseISO
    }
  },
  computed: {
    file () {
      return this.$store.state.overlay.match.file
    }
  },
  mounted () {
    this.initView()
  },
  methods: {
    initView () {
      const commonWords = [
        '180', '180x180', '2880x1440', '3d', '3dh', '3dv', '30fps', '30m', '360',
        '3840x1920', '4k', '5k', '5400x2700', '60fps', '6k', '7k', '7680x3840',
        '8k', 'fb360', 'fisheye190', 'funscript', 'h264', 'h265', 'hevc', 'hq', 'hsp', 'lq', 'lr',
        'mkv', 'mkx200', 'mkx220', 'mono', 'mp4', 'oculus', 'oculus5k',
        'oculusrift', 'original', 'rf52', 'smartphone', 'srt', 'ssa', 'tb', 'uhq', 'vrca220', 'vp9'
      ]
      const isNotCommonWord = word => !commonWords.includes(word.toLowerCase()) && !/^[0-9]+p$/.test(word)

      this.data = []
      this.queryString = (
        this.file.filename
          .replace(/\.|_|\+|-/g, ' ').replace(/\s+/g, ' ').trim()
          .split(' ').filter(isNotCommonWord).join(' ')
          .replace(/ s /g, '\'s '))
      this.loadData()
    },
    loadData: async function loadData () {
      const requestIndex = this.dataNumRequests
      this.dataNumRequests = this.dataNumRequests + 1

      const resp = await ky.get('/api/scene/search', {
        searchParams: {
          q: this.queryString
        },
        timeout: 60000
      }).json()

      if (requestIndex >= this.dataNumResponses) {
        this.dataNumResponses = requestIndex + 1

        if (resp.scenes !== null) {
          this.data = resp.scenes
        } else {
          this.data = []
        }
        this.currentPage = 1
      }
    },
    getImageURL (u) {
      if (u.startsWith('http')) {
        return '/img/120x/' + u.replace('://', ':/')
      } else {
        return u
      }
    },
    assign: async function assign (scene_id) {
      await ky.post('/api/files/match', {
        json: {
          file_id: this.toInt(this.$store.state.overlay.match.file.id),
          scene_id: scene_id
        }
      })

      this.$store.dispatch('files/load')

      const data = this.$store.getters['files/nextFile'](this.file)
      if (data !== null) {
        this.nextFile()
      } else {
        this.close()
      }
    },
    nextFile () {
      const data = this.$store.getters['files/nextFile'](this.file)
      if (data !== null) {
        this.$store.commit('overlay/showMatch', { file: data })
        this.initView()
      }
    },
    prevFile () {
      const data = this.$store.getters['files/prevFile'](this.file)
      if (data !== null) {
        this.$store.commit('overlay/showMatch', { file: data })
        this.initView()
      }
    },
    close () {
      this.$store.commit('overlay/hideMatch')
    },
    toInt (value, radix, defaultValue) {
      return parseInt(value, radix || 10) || defaultValue || 0
    },
    videoFilesCount (scene) {
      let count = 0
      console.log(scene)
      scene.file.forEach(obj => {
        if (obj.type === 'video') {
          count = count + 1
        }
      })
      return count
    },
    scriptFilesCount (scene) {
      let count = 0
      scene.file.forEach(obj => {
        if (obj.type === 'script') {
          count = count + 1
        }
      })
      return count
    },
    subtitlesFilesCount (scene) {
      let count = 0
      scene.file.forEach(obj => {
        if (obj.type === 'subtitles') {
          count = count + 1
        }
      })
      return count
    },
    handleRightArrow () {
      if ((this.currentPage) * 5 < this.data.length) {
        this.currentPage = this.currentPage + 1
      } else {
        this.currentPage = 1
      }
    },
    handleLeftArrow () {
      if (this.currentPage === 1) {
        // dont assume last page is 5
        this.currentPage = ~~((this.data.length + 4) / 5)
      } else {
        this.currentPage = this.currentPage - 1
      }
    },
    searchPrefix(prefix) {
      let textbox = this.$refs.searchInput
      if (textbox.selectionStart != textbox.selectionEnd) {
        let selected = textbox.value.substring(textbox.selectionStart, textbox.selectionEnd)
        selected=selected.replace(/_/g," ").replace(/-/g," ").trim()
        if (selected.indexOf(' ') >= 0)
        {
          selected='"' + selected + '"'
        }        
        this.queryString = textbox.value.substring(0,textbox.selectionStart) + " " + prefix + selected + " " + textbox.value.substr(textbox.selectionEnd)
        this.loadData()
      }
      
    },
    searchDatePrefix(prefix) {      
        let today = new Date().toISOString().slice(0, 10)
        let weekago = new Date(Date.now() - 604800000).toISOString().slice(0, 10)        
          this.queryString = this.queryString.trim() + ' ' + prefix + '>="' + weekago + '" ' +  prefix + '<="' + today + '"'        
        this.loadData()
    },
    prettyBytes
  }
}
</script>

<style scoped>
h6.title.is-6 {
  margin-bottom: 0;
}

h6 + small {
  margin-bottom: 1.5rem;
  display: inline-block;
  font-size: small;
}

h6 + small > .pathDetails {
  color: #B0B0B0;
}

.modal-card {
  position: absolute;
  top: 4em;
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
</style>
