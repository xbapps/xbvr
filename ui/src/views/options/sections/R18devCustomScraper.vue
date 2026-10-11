<template>
  <div class="r18dev-custom">
    <b-field :label="$t('R18.dev list link')" label-position="on-border">
      <b-input v-model="url" :placeholder="$t('A studio, label or category on R18.dev, e.g. https://r18.dev/videos/vod/movies/list/?id=46483&type=studio')"/>
    </b-field>

    <!-- Suggestions: studios we don't ship as presets, one click fills the form -->
    <template v-if="url.trim() === '' && !editing && suggestions.length">
      <p class="help hint">{{ $t('Or start from one of these:') }}</p>
      <div class="cards">
        <div v-for="s in suggestions" :key="s.id" class="card-s">
          <div class="card-top">
            <img v-if="s.avatar_url" :src="getImageURL(s.avatar_url)" class="avatar" @error="$event.target.style.display = 'none'">
            <b>{{ s.name }}</b>
          </div>
          <div>{{ s.scenes.toLocaleString() }} {{ $t('scenes') }}, {{ s.sites_display || (s.sites || []).join(', ') }}</div>
          <div class="state">{{ s.state }}<template v-if="s.note">, {{ s.note }}</template></div>
          <b-button size="is-small" type="is-primary" outlined @click="use(s)">{{ $t('Use') }}</b-button>
        </div>
      </div>
      <p class="help">{{ $t('"Use" fills the form with that scraper\'s link, name, avatar and prefixes. Nothing is saved until Save. Suggestions you already saved disappear.') }}</p>
    </template>

    <template v-if="url.trim() !== ''">
      <b-field :label="$t('More R18.dev list links')" label-position="on-border">
        <b-taginput v-model="extraUrls" :placeholder="$t('Optional: more studio / label / category links, Enter after each')"
                    :confirm-keys="['Enter', 'Tab']" :on-paste-separators="[' ', '\n']" ellipsis/>
      </b-field>

      <div class="previews">
        <div v-for="u in sourceUrls" :key="u" class="preview">
          <template v-if="previews[u] && previews[u].error">
            <span class="has-text-danger">{{ previews[u].error }}</span>
          </template>
          <template v-else-if="previews[u]">
            <img v-if="previews[u].newest.thumb" :src="getImageURL(previews[u].newest.thumb)" class="thumb"
                 @mouseenter="showCover(previews[u].newest.content_id)" @mouseleave="hideCover">
            <span>
              <b>{{ previews[u].type }} {{ previews[u].id }}<template v-if="previews[u].maker">: {{ previews[u].maker }}</template></b><br>
              {{ previews[u].total.toLocaleString() }} {{ $t('scenes') }}<br>
              <template v-if="previews[u].newest.display_id">{{ $t('latest') }} {{ previews[u].newest.display_id }} ({{ previews[u].newest.released }})<br></template>
              <small><span class="has-text-primary">{{ $t('best guess at the newest scene this would scrape') }}</span><br>
              {{ $t('preorders and compilations skipped') }}<br>
              {{ $t('hover for full cover image') }}</small>
            </span>
          </template>
          <span v-else class="has-text-grey">{{ $t('Checking') }} {{ u }} ...</span>
        </div>
      </div>

      <b-field :label="$t('Name')" label-position="on-border">
        <b-input v-model="name" :placeholder="$t('Enter Studio Name')" :disabled="!!editing"/>
      </b-field>
      <p class="help id-line">
        {{ $t('Site ID') }}: <code>r18dev-{{ id || '...' }}</code>
        <span v-if="editing"> ({{ $t('editing, the name stays') }})</span>
      </p>
      <b-field :label="$t('Company')" label-position="on-border">
        <b-input v-model="company" :placeholder="$t('Optional: defaults to Name')"/>
      </b-field>
      <b-field :label="$t('Avatar Url')" label-position="on-border">
        <b-input v-model="avatar" :placeholder="$t('Optional')"/>
      </b-field>
      <b-field :label="$t('DVD prefixes (XBVR &quot;Site&quot;)')" label-position="on-border">
        <b-taginput v-model="sites" :placeholder="$t('Filled in from the newest scenes, e.g. CAFR')" :confirm-keys="[',', 'Enter', 'Tab', ' ']" ellipsis/>
      </b-field>
      <b-field :label="$t('Site column text')" label-position="on-border">
        <b-input v-model="sitesDisplay" :placeholder="$t('Optional, e.g. CAFR (+CACA +CAMI)')"/>
      </b-field>
      <b-field :label="$t('Extra content IDs')" label-position="on-border">
        <b-taginput v-model="contentIds" :placeholder="$t('Optional: scenes no list carries, e.g. 55tmavr00179')" :confirm-keys="[',', 'Enter', 'Tab', ' ']" ellipsis/>
      </b-field>
    </template>

    <div v-if="url.trim() !== '' || editing" class="buttons">
      <b-tooltip :label="$t('Restart XBVR to load new Sites')" :delay="500" type="is-warning">
        <b-button type="is-primary" :disabled="!valid" :loading="saving" @click="save">{{ $t('Save') }}</b-button>
      </b-tooltip>
      <b-button @click="reset">{{ editing ? $t('Cancel') : $t('Clear') }}</b-button>
      <b-button v-if="editing" type="is-danger" outlined @click="remove(editing)">{{ $t('Delete') }}</b-button>
    </div>

    <!-- Saved custom R18.dev Scrapers -->
    <div v-if="list.length || removed.length" class="saved">
      <p class="label-small">{{ $t('Your custom R18.dev scrapers') }}</p>
      <table class="table is-narrow is-hoverable">
        <tbody>
          <tr v-for="s in list" :key="s.id">
            <td><a @click="edit(s)">{{ s.name }}</a></td>
            <td class="has-text-grey"><code>r18dev-{{ s.id }}</code></td>
            <td>{{ s.sites_display || (s.sites || []).join(', ') }}</td>
            <td>
              <b-tag v-if="s.status === 'restart'" type="is-warning" size="is-small">{{ $t('restart XBVR to load') }}</b-tag>
              <b-tag v-else type="is-success" size="is-small" class="is-light">{{ $t('loaded') }}</b-tag>
            </td>
            <td class="actions">
              <a @click="edit(s)" :title="$t('Edit')"><b-icon pack="mdi" icon="pencil" size="is-small"/></a>
              <a @click="remove(s.id)" :title="$t('Delete')"><b-icon pack="mdi" icon="delete" size="is-small"/></a>
            </td>
          </tr>
          <tr v-for="id in removed" :key="'removed-' + id" class="has-text-grey">
            <td><s>{{ id }}</s></td>
            <td><code>r18dev-{{ id }}</code></td>
            <td></td>
            <td><b-tag type="is-warning" size="is-small">{{ $t('deleted, restart XBVR to unload') }}</b-tag></td>
            <td></td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Full cover on hover: hotlinked straight from DMM, never stored -->
    <div v-if="cover" class="cover-overlay">
      <img v-show="coverLoaded" :src="cover" @load="coverLoaded = true" @error="coverFallback">
      <span v-if="!coverLoaded" class="tag is-dark">{{ $t('Loading cover') }} ...</span>
    </div>
  </div>
</template>

<script>
import ky from 'ky'

const r18devListURL = (src) => `https://r18.dev/videos/vod/movies/list/?id=${src.id}&type=${src.type}`

export default {
  name: 'R18devCustomScraper',
  data () {
    return {
      list: [],
      removed: [],
      suggestions: [],
      url: '',
      editing: null, // ID of the saved scraper in the form
      presetId: '', // ID a suggestion comes with
      name: '',
      company: '',
      avatar: '',
      sites: [],
      sitesDisplay: '',
      contentIds: [],
      extraUrls: [],
      previews: {}, // link -> preview, or {error}
      saving: false,
      timer: null,
      cover: '', // full cover URL while hovering a preview thumbnail
      coverCid: '',
      coverLoaded: false
    }
  },
  computed: {
    sourceUrls () {
      return [this.url.trim(), ...this.extraUrls.map(u => u.trim())].filter(u => u !== '')
    },
    sources () {
      return this.sourceUrls.map(u => this.previews[u]).filter(p => p && !p.error).map(p => ({ type: p.type, id: p.id }))
    },
    id () {
      if (this.editing) return this.editing
      if (this.presetId) return this.presetId
      return this.name.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-+|-+$/g, '')
    },
    valid () {
      const allChecked = this.sourceUrls.every(u => this.previews[u] && !this.previews[u].error)
      return this.id !== '' && this.sources.length > 0 && allChecked
    }
  },
  watch: {
    sourceUrls () {
      clearTimeout(this.timer)
      this.timer = setTimeout(this.checkLinks, 600)
    }
  },
  mounted () {
    this.load()
  },
  methods: {
    // awsimgsrc serves the full size jacket (2184x1365 for VR), pics.dmm.co.jp tops out at 800 wide
    showCover (cid) {
      if (!cid) return
      this.coverCid = cid
      this.coverLoaded = false
      this.cover = `https://awsimgsrc.dmm.com/dig/digital/video/${cid}/${cid}pl.jpg`
    },
    coverFallback () {
      const small = `https://pics.dmm.co.jp/digital/video/${this.coverCid}/${this.coverCid}pl.jpg`
      this.cover = this.cover === small ? '' : small
    },
    hideCover () {
      this.cover = ''
      this.coverLoaded = false
    },
    getImageURL (u) {
      return u.startsWith('http') ? '/img/120x/' + u.replace('://', ':/') : u
    },
    load () {
      ky.get('/api/options/r18dev-custom').json().then(data => {
        this.list = data.scrapers
        this.removed = data.removed
        this.suggestions = data.suggestions || []
      })
    },
    checkLinks () {
      for (const u of this.sourceUrls) {
        if (this.previews[u]) continue
        if (!/r18\.dev\//i.test(u)) {
          this.$set(this.previews, u, { error: `${u}: not an R18.dev link` })
          continue
        }
        ky.get('/api/options/r18dev-preview', { searchParams: { url: u }, timeout: 60000 }).json()
          .then(p => {
            this.$set(this.previews, u, p)
            if (!this.editing && !this.presetId) {
              if (this.name === '' && p.type === 'studio' && p.maker) this.name = p.maker
              if (this.sites.length === 0) this.sites = p.prefixes.slice(0, 6)
            }
          })
          .catch(async e => {
            const msg = e.response ? await e.response.text() : e.message
            this.$set(this.previews, u, { error: `${u}: ${msg}` })
          })
      }
    },
    save () {
      this.saving = true
      ky.put('/api/options/r18dev-custom', {
        json: {
          editing: !!this.editing,
          config: {
            id: this.id,
            name: this.name,
            company: this.company,
            avatar_url: this.avatar,
            sources: this.sources,
            sites: this.sites,
            sites_display: this.sitesDisplay,
            content_ids: this.contentIds
          }
        }
      }).json().then(saved => {
        this.$buefy.toast.open({ message: `${saved.name} saved as r18dev-${saved.id}. Restart XBVR to load it.`, type: 'is-success', duration: 8000 })
        this.reset()
        this.load()
      }).catch(async e => {
        const msg = e.response ? await e.response.text() : e.message
        this.$buefy.toast.open({ message: msg, type: 'is-danger', duration: 10000 })
      }).finally(() => {
        this.saving = false
      })
    },
    fill (s) {
      this.name = s.name
      this.company = s.company === s.name ? '' : (s.company || '')
      this.avatar = s.avatar_url || ''
      this.sites = [...(s.sites || [])]
      this.sitesDisplay = s.sites_display || ''
      this.contentIds = [...(s.content_ids || [])]
      const links = (s.sources || []).map(r18devListURL)
      this.extraUrls = links.slice(1)
      this.url = links[0] || ''
    },
    use (s) {
      this.editing = null
      this.presetId = s.id
      this.fill(s)
    },
    edit (s) {
      this.presetId = ''
      this.editing = s.id
      this.fill(s)
    },
    reset () {
      this.editing = null
      this.presetId = ''
      this.name = ''
      this.company = ''
      this.avatar = ''
      this.sites = []
      this.sitesDisplay = ''
      this.contentIds = []
      this.extraUrls = []
      this.url = ''
    },
    remove (id) {
      this.$buefy.dialog.confirm({
        title: this.$t('Delete custom scraper'),
        message: `Delete <b>r18dev-${id}</b>? Its scenes stay in XBVR. After a restart, whichever scraper lists them next takes them over.`,
        type: 'is-danger',
        confirmText: this.$t('Delete'),
        onConfirm: () => {
          ky.delete(`/api/options/r18dev-custom/${encodeURIComponent(id)}`).then(() => {
            if (this.editing === id) this.reset()
            this.load()
          })
        }
      })
    }
  }
}
</script>

<style scoped>
.hint {
  margin: -0.6rem 0 0.4rem;
}
.cards {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 0.5rem;
  margin-bottom: 0.3rem;
}
.card-s {
  border: 1px solid #e5e5e5;
  border-radius: 4px;
  padding: 0.5rem;
  display: flex;
  flex-direction: column;
  gap: 0.2rem;
  font-size: 0.8rem;
  color: #4a4a4a;
}
.card-s .button {
  align-self: flex-start;
}
.card-top {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.85rem;
  color: #363636;
}
.avatar {
  width: 30px;
  height: 30px;
  object-fit: contain;
}
.state {
  color: #a15c12;
}
.previews {
  margin: -0.5rem 0 0.75rem;
  font-size: 0.85rem;
}
.preview {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  min-height: 1.5rem;
  margin-bottom: 0.25rem;
}
.thumb {
  width: 120px;
  height: auto;
  cursor: zoom-in;
}
.cover-overlay {
  position: fixed;
  inset: 0;
  z-index: 50;
  display: flex;
  align-items: center;
  justify-content: center;
  pointer-events: none;
}
/* the size of the scene details modal, never past the image's own size */
.cover-overlay img {
  max-width: 85vw;
  max-height: 85vh;
  box-shadow: 0 8px 32px rgba(10, 10, 10, 0.5);
}
.id-line {
  margin: -0.6rem 0 0.75rem;
}
.saved {
  margin-top: 1rem;
}
.label-small {
  font-weight: 600;
  margin-bottom: 0.25rem;
}
.saved .table {
  font-size: 0.85rem;
}
.saved .actions a {
  margin-right: 0.25rem;
}
</style>
