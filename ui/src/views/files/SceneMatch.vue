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

          <b-field :label="$t('Match context')" label-position="on-border" class="match-context"
                   :message="contextDirty ? $t('Not saved yet') : $t('Saved with this file. Added to the search below, to web searches, and to what the LLM is told.')">
            <b-input v-model="matchContext" expanded
                     :placeholder="$t('e.g. studio, performer names, a better title')"
                     @keyup.native.enter="saveContext"></b-input>
            <p class="control">
              <b-button :type="contextDirty ? 'is-primary' : 'is-light'" @click="saveContext" :loading="savingContext">{{ $t('Save') }}</b-button>
            </p>
          </b-field>

          <b-field grouped>
            <b-taglist>
              <b-tag class="tag is-info is-small">{{$t('Search Fields')}}</b-tag>
              <b-tooltip :label="$t('Optional: select one or more words to target searching to a specific field')" :delay="500" position="is-top">
                <b-button @click='searchPrefix("+title:")' class="tag is-info is-small is-light">title:</b-button>
                <b-button @click='searchPrefix("cast:")' class="tag is-info is-small is-light">cast:</b-button>
                <b-button @click='searchPrefix("+site:")' class="tag is-info is-small is-light">site:</b-button>
                <b-button @click='searchPrefix("+id:")' class="tag is-info is-small is-light">id:</b-button>
              </b-tooltip>&nbsp;
              <b-tooltip :label="$t('Add file duration to search')" :delay="500" position="is-top">
                <b-button @click='searchDurationPrefix("duration:")' class="tag is-info is-small is-light">duration:</b-button>
              </b-tooltip>&nbsp;
              <b-tooltip :label="$t('Defaults date range to the last week. Note:must match yyyy-mm-dd, include leading zeros')" :delay="500" position="is-top">
                <b-button @click='searchDatePrefix("released:")' class="tag is-info is-small is-light">released:</b-button>
                <b-button @click='searchDatePrefix("added:")' class="tag is-info is-small is-light">added:</b-button>
              </b-tooltip>
            </b-taglist>          
          </b-field>
          <b-field :label="$t('Search')" label-position="on-border">
            <div class="control">
              <input class="input" type="text" v-model='queryString' v-debounce:200ms="loadData" autofocus ref="searchInput">
            </div>
          </b-field>
          
          <b-table :data="data" ref="table" paginated :current-page.sync="currentPage" per-page="5" :default-sort="['_score', 'desc']">
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
            <b-table-column field="duration" :label="$t('Duration')" sortable nowrap v-slot="props">
              {{ props.row.duration > 0 ? props.row.duration + " min" : ""}}
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

          <div class="drafts">
            <h6 class="title is-6">{{ $t('Draft scenes') }}
              <small class="has-text-grey">&mdash; {{ $t('proposed from web pages by the LLM scraper; nothing is added until you save one') }}</small>
            </h6>
            <b-field grouped group-multiline>
              <p class="control">
                <b-button type="is-info" outlined icon-left="magnify" @click="suggestDrafts" :loading="suggesting" :disabled="scrapingUrl">
                  {{ $t('Search the web') }}
                </b-button>
              </p>
              <b-input v-model="scrapeUrl" expanded :placeholder="$t('...or paste the URL of the scene page')" @keyup.native.enter="scrapeOneUrl"></b-input>
              <p class="control">
                <b-button @click="scrapeOneUrl" :loading="scrapingUrl" :disabled="!scrapeUrl || suggesting">{{ $t('Read page') }}</b-button>
              </p>
            </b-field>
            <p v-if="suggesting" class="has-text-grey"><small>{{ $t('Searching and reading pages; this usually takes under a minute.') }}</small></p>
            <b-notification v-if="lastRunNotes.length" type="is-light" :closable="true" @close="lastRunNotes = []" class="run-notes">
              <p v-if="lastQuery"><small>{{ $t('Searched for') }} <code>{{ lastQuery }}</code>
                <span v-if="lastKnownDownloadSites">&mdash; {{ lastKnownDownloadSites }} {{ $t('results from known download sites skipped') }}</span></small></p>
              <p v-for="(n, idx) in lastRunNotes" :key="idx"><small>
                <a :href="n.url" target="_blank" rel="noreferrer">{{ shortUrl(n.url) }}</a>:
                <span :class="n.error ? 'has-text-danger' : 'has-text-grey'">{{ n.error || n.skipped }}</span>
              </small></p>
            </b-notification>

            <b-table v-if="drafts.length" :data="drafts" :row-class="draftRowClass">
              <b-table-column field="cover" :label="$t('Image')" width="120" v-slot="props">
                <vue-load-image>
                  <img slot="image" :src="getImageURL(draftCover(props.row))"/>
                  <img slot="preloader" src="/ui/images/blank.png"/>
                  <img slot="error" src="/ui/images/blank.png"/>
                </vue-load-image>
              </b-table-column>
              <b-table-column field="site" :label="$t('Site')" v-slot="props">
                <a :href="props.row.source_url" target="_blank" rel="noreferrer">{{ props.row.scene.site }}</a><br/>
                <b-tag size="is-small" :type="pageKindType(props.row.page_kind)">{{ pageKindLabel(props.row.page_kind) }}</b-tag>
              </b-table-column>
              <b-table-column field="title" :label="$t('Title')" v-slot="props">
                <p>{{ props.row.scene.title }}</p>
                <small><b-tag rounded v-for="c in (props.row.scene.cast || [])" :key="c">{{ c }}</b-tag></small>
              </b-table-column>
              <b-table-column field="released" :label="$t('Release date')" nowrap v-slot="props">
                {{ props.row.scene.released }}
              </b-table-column>
              <b-table-column field="duration" :label="$t('Duration')" nowrap v-slot="props">
                {{ props.row.scene.duration > 0 ? props.row.scene.duration + " min" : "" }}
              </b-table-column>
              <b-table-column field="match_confidence" :label="$t('Confidence')" v-slot="props">
                <div :title="props.row.reason" class="confidence">
                  <b-progress show-value :value="props.row.match_confidence * 100"
                              :type="props.row.match_confidence >= minConfidence ? 'is-success' : 'is-warning'"></b-progress>
                </div>
              </b-table-column>
              <b-table-column field="_actions" v-slot="props">
                <div class="buttons are-small is-flex-wrap-nowrap">
                  <b-button @click="openPreview(props.row)">{{ $t('Preview') }}</b-button>
                  <b-button type="is-primary" outlined @click="saveDraft(props.row)" :loading="savingDraft === props.row.id">{{ $t('Save') }}</b-button>
                  <b-button type="is-danger" outlined icon-left="close" @click="rejectDraft(props.row)" :title="$t('Reject: never propose this page for this file again')"></b-button>
                </div>
              </b-table-column>
            </b-table>
            <p v-else-if="!suggesting && !scrapingUrl" class="has-text-grey"><small>{{ $t('No drafts for this file yet.') }}</small></p>
          </div>
        </div>
      </section>
    </div>

    <b-modal v-model="previewOpen" has-modal-card :can-cancel="['outside', 'x']">
      <div class="modal-card draft-preview" v-if="preview">
        <header class="modal-card-head">
          <p class="modal-card-title">{{ preview.scene.title || $t('Untitled draft') }}</p>
          <button class="delete" @click="previewOpen = false" aria-label="close"></button>
        </header>
        <section class="modal-card-body">
          <div class="columns">
            <div class="column is-half">
              <img v-if="draftCover(preview)" :src="getImageURL(draftCover(preview), 700)" class="preview-cover"/>
              <p v-else class="has-text-grey">{{ $t('No cover image') }}</p>
              <div class="preview-gallery" v-if="preview.scene.gallery && preview.scene.gallery.length">
                <a v-for="g in preview.scene.gallery" :key="g" :href="g" target="_blank" rel="noreferrer">
                  <img :src="getImageURL(g, 200)"/>
                </a>
              </div>
            </div>
            <div class="column">
              <table class="table is-narrow is-fullwidth">
                <tbody>
                  <tr><th>{{ $t('Site') }}</th><td>{{ preview.scene.site }}</td></tr>
                  <tr v-if="preview.scene.studio !== preview.scene.site"><th>{{ $t('Studio') }}</th><td>{{ preview.scene.studio }}</td></tr>
                  <tr><th>{{ $t('Cast') }}</th><td><b-tag rounded v-for="c in (preview.scene.cast || [])" :key="c">{{ c }}</b-tag></td></tr>
                  <tr><th>{{ $t('Released') }}</th><td>{{ preview.scene.released || '—' }}</td></tr>
                  <tr><th>{{ $t('Duration') }}</th><td>{{ preview.scene.duration > 0 ? preview.scene.duration + ' min' : '—' }}
                    <small v-if="file.duration > 0" class="has-text-grey">({{ $t('file') }}: {{ Math.floor(file.duration / 60) }} min)</small></td></tr>
                  <tr><th>{{ $t('Scene ID') }}</th><td><code>{{ preview.scene._id }}</code></td></tr>
                  <tr><th>{{ $t('Source') }}</th><td><a :href="preview.scene.homepage_url" target="_blank" rel="noreferrer">{{ shortUrl(preview.scene.homepage_url) }}</a>
                    <b-tag size="is-small" :type="pageKindType(preview.page_kind)">{{ pageKindLabel(preview.page_kind) }}</b-tag></td></tr>
                  <tr v-if="preview.scene.trailer_source"><th>{{ $t('Trailer') }}</th><td><a :href="preview.scene.trailer_source" target="_blank" rel="noreferrer">{{ shortUrl(preview.scene.trailer_source) }}</a></td></tr>
                  <tr><th>{{ $t('Confidence') }}</th><td>{{ Math.round(preview.match_confidence * 100) }}% &mdash; <small>{{ preview.reason }}</small></td></tr>
                </tbody>
              </table>
              <p v-if="preview.scene.synopsis" class="preview-synopsis">{{ preview.scene.synopsis }}</p>
              <b-taglist v-if="preview.scene.tags && preview.scene.tags.length">
                <b-tag v-for="t in preview.scene.tags" :key="t" size="is-small">{{ t }}</b-tag>
              </b-taglist>
            </div>
          </div>
        </section>
        <footer class="modal-card-foot">
          <b-button type="is-primary" @click="saveDraft(preview)" :loading="savingDraft === preview.id">{{ $t('Save and assign to this file') }}</b-button>
          <b-button type="is-danger" outlined @click="rejectDraft(preview)">{{ $t('Reject') }}</b-button>
          <b-button @click="previewOpen = false">{{ $t('Close') }}</b-button>
        </footer>
      </div>
    </b-modal>
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
      baseQuery: '',
      matchContext: '',
      savedContext: '',
      savingContext: false,
      drafts: [],
      suggesting: false,
      scrapingUrl: false,
      scrapeUrl: '',
      lastQuery: '',
      lastKnownDownloadSites: 0,
      lastRunNotes: [],
      previewOpen: false,
      preview: null,
      savingDraft: 0,
      minConfidence: 0.5,
      format,
      parseISO
    }
  },
  computed: {
    file () {
      return this.$store.state.overlay.match.file
    },
    contextDirty () {
      return this.matchContext.trim() !== this.savedContext
    }
  },
  mounted () {
    this.initView()
    ky.get('/api/llmscrape/config').json()
      .then(cfg => { this.minConfidence = cfg.minConfidence })
      .catch(() => {})
  },
  methods: {
    initView () {
      const commonWords = [
        '180', '180x180', '2880x1440', '3d', '3dh', '3dv', '30fps', '30m', '360',
        '3840x1920', '4k', '5k', '5400x2700', '60fps', '6k', '7k', '7680x3840',
        '8k', 'fb360', 'fisheye190', 'funscript', 'cmscript', 'h264', 'h265', 'hevc', 'hq', 'hsp', 'lq', 'lr',
        'mkv', 'mkx200', 'mkx220', 'mono', 'mp4', 'oculus', 'oculus5k',
        'oculusrift', 'original', 'rf52', 'smartphone', 'srt', 'ssa', 'tb', 'uhq', 'vrca220', 'vp9'
      ]
      const isNotCommonWord = word => !commonWords.includes(word.toLowerCase()) && !/^[0-9]+p$/.test(word)

      this.data = []
      this.drafts = []
      this.lastRunNotes = []
      this.scrapeUrl = ''
      this.previewOpen = false
      this.baseQuery = (
        this.file.filename
          .replace(/[._+'’`-]/g, ' ').replace(/\s+/g, ' ').trim()
          .split(' ').filter(isNotCommonWord).join(' '))
      this.matchContext = ''
      this.savedContext = ''
      this.queryString = this.baseQuery
      this.loadData()
      this.loadContext()
      this.loadDrafts()
    },
    // Responses can arrive after the user has moved to another file; only apply them to the file
    // they were requested for.
    stillOn (fileId) {
      return this.file && this.file.id === fileId
    },
    async loadContext () {
      const fileId = this.file.id
      try {
        const r = await ky.get(`/api/llmscrape/context/${fileId}`).json()
        if (!this.stillOn(fileId)) return
        if (r.folder_terms) {
          // The folder holds only this video, so its name describes the scene.
          this.baseQuery = this.withoutRepeats(r.folder_terms + ' ' + this.baseQuery)
        }
        if (r.context) {
          this.matchContext = r.context
          this.savedContext = r.context
        }
        if (r.folder_terms || r.context) {
          this.queryString = (this.baseQuery + ' ' + (r.context || '')).trim()
          this.loadData()
        }
      } catch (e) {
        // The match screen works without the LLM scraper.
      }
    },
    withoutRepeats (text) {
      const seen = new Set()
      return text.split(/\s+/).filter(w => {
        const k = w.toLowerCase()
        if (!w || seen.has(k)) return false
        seen.add(k)
        return true
      }).join(' ')
    },
    async saveContext () {
      const fileId = this.file.id
      const context = this.matchContext.trim()
      this.savingContext = true
      try {
        await ky.post(`/api/llmscrape/context/${fileId}`, { json: { context } })
        if (!this.stillOn(fileId)) return
        this.savedContext = context
        this.matchContext = context
        this.queryString = (this.baseQuery + ' ' + context).trim()
        this.loadData()
      } catch (e) {
        this.toastError(e)
      } finally {
        this.savingContext = false
      }
    },
    async loadDrafts () {
      const fileId = this.file.id
      try {
        const drafts = await ky.get('/api/llmscrape/drafts', {
          searchParams: { file_id: fileId, status: 'draft' }
        }).json()
        if (this.stillOn(fileId)) this.drafts = drafts
      } catch (e) {
        if (this.stillOn(fileId)) this.drafts = []
      }
    },
    async suggestDrafts () {
      const fileId = this.file.id
      if (this.contextDirty) await this.saveContext()
      this.suggesting = true
      this.lastRunNotes = []
      try {
        // Replaces this file's unreviewed search drafts, so they are judged again with the
        // current context; rejected drafts and pasted URLs stay.
        const r = await ky.post(`/api/llmscrape/suggest/${fileId}`, { searchParams: { refresh: true }, timeout: 600000 }).json()
        if (!this.stillOn(fileId)) return
        this.lastQuery = r.query
        this.lastKnownDownloadSites = r.known_download_sites || 0
        this.lastRunNotes = (r.results || []).filter(x => !x.draft)
        if (!(r.results || []).length && !this.lastKnownDownloadSites) {
          this.lastRunNotes = [{ url: '', skipped: this.$t('No new pages found. Try adding context, or paste a URL.') }]
        }
        await this.loadDrafts()
      } catch (e) {
        this.toastError(e)
      } finally {
        this.suggesting = false
      }
    },
    async scrapeOneUrl () {
      if (!this.scrapeUrl) return
      const fileId = this.file.id
      this.scrapingUrl = true
      try {
        const r = await ky.post('/api/llmscrape/scrape', {
          json: { url: this.scrapeUrl.trim(), file_id: fileId },
          timeout: 300000
        }).json()
        if (!this.stillOn(fileId)) return
        if (r.error || r.skipped) {
          this.lastQuery = ''
          this.lastRunNotes = [r]
        } else {
          this.scrapeUrl = ''
          await this.loadDrafts()
          const made = this.drafts.find(d => d.id === r.draft.id)
          if (made) this.openPreview(made)
        }
      } catch (e) {
        this.toastError(e)
      } finally {
        this.scrapingUrl = false
      }
    },
    openPreview (draft) {
      this.preview = draft
      this.previewOpen = true
    },
    async saveDraft (draft) {
      this.savingDraft = draft.id
      try {
        await ky.post(`/api/llmscrape/draft/${draft.id}/save`, {
          json: { file_id: this.toInt(this.file.id) },
          timeout: 120000
        }).json()
        this.previewOpen = false
        this.$buefy.toast.open({ message: this.$t('Scene saved and assigned'), type: 'is-success' })
        this.afterAssign()
      } catch (e) {
        this.toastError(e)
      } finally {
        this.savingDraft = 0
      }
    },
    async rejectDraft (draft) {
      try {
        await ky.delete(`/api/llmscrape/draft/${draft.id}`)
        this.drafts = this.drafts.filter(d => d.id !== draft.id)
        if (this.preview && this.preview.id === draft.id) this.previewOpen = false
      } catch (e) {
        this.toastError(e)
      }
    },
    draftCover (draft) {
      const covers = draft.scene.covers
      return covers && covers.length ? covers[0] : ''
    },
    draftRowClass (row) {
      return row.match_confidence >= this.minConfidence ? '' : 'low-confidence'
    },
    pageKindLabel (kind) {
      return {
        official_studio: 'official site',
        store_or_aggregator: 'store',
        download_or_piracy: 'download site',
        forum_or_review: 'forum',
        other: 'other'
      }[kind] || kind
    },
    pageKindType (kind) {
      return kind === 'official_studio' ? 'is-success is-light' : kind === 'download_or_piracy' ? 'is-danger is-light' : 'is-light'
    },
    shortUrl (u) {
      try {
        const x = new URL(u)
        const path = x.pathname.length > 40 ? x.pathname.slice(0, 40) + '…' : x.pathname
        return x.hostname.replace(/^www\./, '') + path
      } catch (e) {
        return u
      }
    },
    async toastError (e) {
      let message = e.message
      if (e.response) {
        const body = await e.response.json().catch(() => null)
        if (body && body.error) message = body.error
      }
      this.$buefy.toast.open({ message, type: 'is-danger', duration: 8000 })
    },
    loadData: async function loadData () {
      const requestIndex = this.dataNumRequests
      this.dataNumRequests = this.dataNumRequests + 1

      const resp = await ky.get('/api/scene/search', {
        searchParams: {
          q: this.queryString,
          fileId: this.toInt(this.file.id)
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
    getImageURL (u, size) {
      if (!u) return ''
      if (u.startsWith('http')) {
        return '/img/' + (size || 120) + 'x/' + u.replace('://', ':/')
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
      this.afterAssign()
    },
    afterAssign () {
      this.$store.dispatch('files/load')

      const data = this.$store.getters['files/nextFile'](this.file)
      if (data !== null) {
        this.nextFile()
      } else {
        this.close()
      }
    },
    nextFile () {
      if (this.previewOpen) return
      const data = this.$store.getters['files/nextFile'](this.file)
      if (data !== null) {
        this.$store.commit('overlay/showMatch', { file: data })
        this.initView()
      }
    },
    prevFile () {
      if (this.previewOpen) return
      const data = this.$store.getters['files/prevFile'](this.file)
      if (data !== null) {
        this.$store.commit('overlay/showMatch', { file: data })
        this.initView()
      }
    },
    close () {
      if (this.previewOpen) {
        this.previewOpen = false
        return
      }
      this.$store.commit('overlay/hideMatch')
    },
    toInt (value, radix, defaultValue) {
      return parseInt(value, radix || 10) || defaultValue || 0
    },
    videoFilesCount (scene) {
      let count = 0      
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
      if (this.previewOpen) return
      if ((this.currentPage) * 5 < this.data.length) {
        this.currentPage = this.currentPage + 1
      } else {
        this.currentPage = 1
      }
    },
    handleLeftArrow () {
      if (this.previewOpen) return
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
    searchDurationPrefix(prefix) {        
        if (this.file.duration==0) {
          this.queryString = this.queryString.trim() + ' ' + prefix + '>=0 '
        } else {
          this.queryString = this.queryString.trim() + ' ' + prefix + '>=' + (Math.floor(this.file.duration / 60)-1) + ' ' +  prefix + '<=' + (Math.floor(this.file.duration / 60)+1) + ''        
        }
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

.match-context {
  margin-bottom: 1.25rem;
}

.drafts {
  margin-top: 2rem;
}

.drafts h6 small {
  font-weight: normal;
  font-size: small;
}

.confidence {
  min-width: 90px;
}

.run-notes {
  padding: 0.75rem 2.5rem 0.75rem 1rem;
}

.drafts >>> tr.low-confidence td {
  opacity: 0.6;
}

.draft-preview.modal-card {
  position: relative;
  top: 0;
  width: 90vw;
  max-width: 1100px;
}

.preview-cover {
  width: 100%;
  border-radius: 4px;
}

.preview-gallery {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  margin-top: 6px;
}

.preview-gallery img {
  height: 70px;
  border-radius: 3px;
}

.preview-synopsis {
  white-space: pre-line;
  margin-bottom: 0.75rem;
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
