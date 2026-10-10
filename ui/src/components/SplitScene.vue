<template>
  <div class="modal-card split-card">
    <header class="modal-card-head split-mhead">
      <p class="modal-card-title">
        <b-icon pack="mdi" icon="call-split" custom-class="split-green"/> Split scene
      </p>
      <button class="delete" aria-label="close" @click="$emit('close')"></button>
    </header>

    <section class="modal-card-body split-body">
      <b-loading :is-full-page="false" :active="loading"></b-loading>
      <template v-if="scene">
        <div class="split-head">
          <img v-if="scene.cover_url" :src="imageURL(scene.cover_url)" class="split-cover">
          <div>
            <p class="has-text-weight-bold split-title">{{ scene.title }}</p>
            <p class="is-size-7">{{ castNames }} &middot; {{ scene.studio }} &middot; {{ scene.duration }} min &middot; {{ rows.length }} filenames, {{ matchedCount }} matched files</p>
            <div class="split-parts-row">
              <label class="label">Number of parts</label>
              <b-select size="is-small" v-model.number="partCount" @input="resetDefaults">
                <option v-for="n in partChoices" :key="n" :value="n">{{ n }}</option>
              </b-select>
              <span v-for="n in partCount" :key="'p' + n" class="split-pill" :class="{ on: enabled[n] }">
                <input type="checkbox" v-model="enabled[n]" @change="resetDefaults"> _split{{ n }}
              </span>
            </div>
            <p class="is-size-7 has-text-grey">Untick a part to skip its number (part 1 + part 3 is fine). Changing parts resets the choices below.</p>
          </div>
        </div>

        <article class="message is-warning is-small split-warn"><div class="message-body"><ul>
          <li><b>Edits already made to {{ scene.scene_id }} are copied to every split and re-applied.</b> Edits made after the split only change the split they were made on.</li>
          <li><b>Duration</b> is added up from the matched video files of each split. Check it; names with no file behind them add nothing.</li>
          <li>Each filename can only belong to one scene, or XBVR can't match it to either. Whatever stays in <b>none</b> stays with {{ scene.scene_id }}.</li>
          <li><b>Favorite, watched and rating</b> start out like {{ scene.scene_id }} and can be changed per split. Hidden starts off. Cuepoints, watch history, StashDB links and the multipart flag are not copied.</li>
        </ul></div></article>

        <p class="split-label">Filenames of {{ scene.scene_id }} and its matched files <span class="split-hint">&#8597; scroll</span></p>
        <div class="split-pane split-files">
          <table class="table is-fullwidth is-narrow is-hoverable">
            <thead><tr>
              <th>File</th><th>Length</th>
              <th v-for="n in parts" :key="'h' + n" class="split-col">_split{{ n }}</th>
              <th class="split-col">none</th><th></th>
            </tr></thead>
            <tbody>
              <template v-for="g in groups">
                <tr :key="'v' + g.video.name" :class="['split-video', { 'split-nameonly': !g.video.file }]">
                  <td><b-icon pack="mdi" :icon="g.video.file ? 'file-video-outline' : 'file-hidden'" size="is-small"/> {{ g.video.name }}</td>
                  <td class="split-dur"><span v-if="g.video.file">{{ fmtDur(g.video.file.duration) }}</span><span v-else class="tag is-light">name only</span>
                    <span v-if="g.video.pending" class="tag is-info is-light" title="A file with this name exists but isn't matched to any scene. The split matches it.">not matched yet</span></td>
                  <td v-for="n in parts" :key="'vr' + n" class="split-col"><input type="radio" :checked="assign[g.video.name] === n" @change="setAssign(g.video.name, n)"></td>
                  <td class="split-col"><input type="radio" :checked="assign[g.video.name] === 0" @change="setAssign(g.video.name, 0)"></td>
                  <td><span v-if="assign[g.video.name] === null" class="tag is-danger is-light">pick a split</span></td>
                </tr>
                <tr v-for="s in g.sides" :key="'s' + s.name" class="split-side">
                  <td class="split-inset"><b-icon pack="mdi" :icon="sideIcon(s.name)" size="is-small"/> {{ s.name }}</td>
                  <td></td>
                  <td v-for="n in parts" :key="'sr' + n" class="split-col" :class="{ follow: isFollowing(s) }"><input type="radio" :disabled="isFollowing(s)" :checked="effective(s.name) === n" @change="setAssign(s.name, n)"></td>
                  <td class="split-col" :class="{ follow: isFollowing(s) }"><input type="radio" :disabled="isFollowing(s)" :checked="effective(s.name) === 0" @change="setAssign(s.name, 0)"></td>
                  <td>
                    <span v-if="s.locked" class="tag is-light" :title="'XBVR matches this file through ' + s.locked + ', so it goes wherever that goes'"><b-icon pack="mdi" icon="lock-outline" size="is-small"/>&nbsp;matches via .mp4</span>
                    <a v-else-if="!(s.name in override)" class="tag is-light" @click="startOverride(s)" title="Assign this file yourself"><b-icon pack="mdi" icon="link-variant" size="is-small"/>&nbsp;follows video</a>
                    <a v-else class="tag is-warning is-light" @click="stopOverride(s)" title="Follow the video again">overridden&nbsp;<b-icon pack="mdi" icon="undo" size="is-small"/></a>
                  </td>
                </tr>
              </template>
              <tr v-if="loose.length" class="split-loosehead"><td :colspan="parts.length + 4">Not matched to one video</td></tr>
              <tr v-for="s in loose" :key="'l' + s.name" class="split-side">
                <td><b-icon pack="mdi" :icon="sideIcon(s.name)" size="is-small"/> {{ s.name }}</td>
                <td></td>
                <td v-for="n in parts" :key="'lr' + n" class="split-col"><input type="radio" :checked="assign[s.name] === n" @change="setAssign(s.name, n)"></td>
                <td class="split-col"><input type="radio" :checked="assign[s.name] === 0" @change="setAssign(s.name, 0)"></td>
                <td><span v-if="assign[s.name] === null" class="tag is-danger is-light">pick a split</span></td>
              </tr>
              <tr class="split-add"><td :colspan="parts.length + 4">
                <b-field grouped>
                  <b-input size="is-small" v-model="newName" placeholder="Add a filename XBVR should match, e.g. 84vrkm00139vrv1uhqe4.mp4" expanded @keyup.native.enter="addName"></b-input>
                  <b-button size="is-small" @click="addName">Add</b-button>
                </b-field>
              </td></tr>
            </tbody>
          </table>
        </div>

        <p class="split-label">New scenes <span class="split-hint">&#8596; scroll</span></p>
        <div class="split-pane split-cards">
          <div v-for="n in parts" :key="'c' + n" class="split-scard">
            <div class="split-trow">
              <p class="split-cid"><b>{{ scene.scene_id }}<span class="split-green">_split{{ n }}</span></b> <span class="tag is-small split-tag">split javr</span></p>
              <star-rating v-model="cards[n].star_rating" :rating="cards[n].star_rating" :increment="0.5" :star-size="16" :show-rating="false"/>
              <div class="buttons are-small">
                <a :class="['button', 'is-small', 'is-danger', { 'is-outlined': !cards[n].favourite }]" @click="cards[n].favourite = !cards[n].favourite" title="Favorite">
                  <b-icon pack="mdi" :icon="cards[n].favourite ? 'heart' : 'heart-outline'" size="is-small"/></a>
                <a :class="['button', 'is-small', 'is-info', { 'is-outlined': !cards[n].watched }]" @click="cards[n].watched = !cards[n].watched" title="Watched">
                  <b-icon pack="mdi" icon="eye" size="is-small"/></a>
                <a :class="['button', 'is-small', 'is-warning', { 'is-outlined': !cards[n].hidden }]" @click="cards[n].hidden = !cards[n].hidden" title="Hidden">
                  <b-icon pack="mdi" icon="eye-off" size="is-small"/></a>
              </div>
            </div>
            <label class="label is-small">Title</label>
            <input class="input is-small" type="text" v-model="cards[n].title">
            <div class="split-crow">
              <div>
                <label class="label is-small">Minutes</label>
                <input class="input is-small" type="number" min="0" :value="minutes(n)" @input="cards[n].duration = $event.target.value" style="width:80px">
                <p class="help">{{ cards[n].duration === null ? 'from ' + videosIn(n).length + ' video' + (videosIn(n).length === 1 ? '' : 's') : 'set by hand' }}</p>
              </div>
              <div class="split-will">
                <label class="label is-small">Will match</label>
                <code v-for="f in shownNamesIn(n)" :key="f">{{ f }}</code>
                <p v-if="lockedIn(n)" class="help">+ {{ lockedIn(n) }} file{{ lockedIn(n) === 1 ? '' : 's' }} via .mp4</p>
                <p v-if="!namesIn(n).length" class="help has-text-danger">nothing yet</p>
              </div>
            </div>
          </div>
        </div>
      </template>
    </section>

    <footer class="modal-card-foot split-foot">
      <div v-if="scene">
        <b>{{ scene.scene_id }} after the split:</b>
        <label class="split-orig"><input type="radio" value="hide" v-model="original"> Hide it</label>
        <label class="split-orig"><input type="radio" value="delete" v-model="original"> Delete it</label>
        <span class="split-permanent">This makes your split permanent</span>
        <p class="help">Hide keeps the "none" filenames, and the split can be undone later. Delete works like Delete Scene; "none" filenames stop matching.</p>
      </div>
      <div>
        <button class="button" @click="$emit('close')">Cancel</button>
        <button class="button split-go" :disabled="!canSplit || busy" @click="doSplit">
          <b-icon pack="mdi" icon="call-split" size="is-small"/><span>Split into {{ parts.length }} scene{{ parts.length === 1 ? '' : 's' }}</span>
        </button>
      </div>
    </footer>
  </div>
</template>

<script>
import ky from 'ky'
import StarRating from 'vue-star-rating'

// Extensions XBVR also matches through the same name with .mp4 (pkg/tasks/volume.go)
const SWAP_EXT = ['funscript', 'hsp', 'srt', 'cmscript']
// Everything that isn't a video
const SIDE_EXT = SWAP_EXT.concat(['ssa', 'ass', 'json', 'txt'])

const natural = (a, b) => a.localeCompare(b, undefined, { numeric: true, sensitivity: 'base' })
const ext = name => { const i = name.lastIndexOf('.'); return i < 0 ? '' : name.slice(i + 1).toLowerCase() }
const stem = name => { const i = name.lastIndexOf('.'); return (i < 0 ? name : name.slice(0, i)).toLowerCase() }
const commonPrefix = (a, b) => { let i = 0; while (i < a.length && i < b.length && a[i] === b[i]) i++; return i }

export default {
  name: 'SplitScene',
  components: { StarRating },
  props: { item: Object },
  data () {
    return {
      loading: true,
      busy: false,
      scene: null,
      extraNames: [],
      unmatched: {}, // filename -> unmatched file with exactly that name
      newName: '',
      partCount: 2,
      enabled: {},
      assign: {}, // filename -> part number, 0 = none, null = not decided
      override: {}, // supplemental filenames assigned by hand instead of following their video
      cards: {},
      original: 'hide'
    }
  },
  async mounted () {
    try {
      this.scene = await ky.get(`/api/scene/${this.item.id}`).json()
      await this.lookupUnmatched(this.rows.filter(r => !r.file).map(r => r.name))
    } catch (e) {
      this.$buefy.toast.open({ message: 'Could not load the scene', type: 'is-danger' })
      this.$emit('close')
      return
    }
    const videos = this.rows.filter(r => r.video)
    const matched = videos.filter(r => r.file && !this.isFiller(r))
    this.partCount = Math.min(Math.max(matched.length || videos.filter(r => !this.isFiller(r)).length, 2), 12)
    this.resetDefaults()
    this.loading = false
  },
  computed: {
    partChoices () {
      return Array.from({ length: 11 }, (_, i) => i + 2)
    },
    parts () {
      const out = []
      for (let n = 1; n <= this.partCount; n++) {
        if (this.enabled[n]) out.push(n)
      }
      return out
    },
    files () {
      return (this.scene && this.scene.file) || []
    },
    matchedCount () {
      return this.files.length
    },
    castNames () {
      return ((this.scene && this.scene.cast) || []).map(c => c.name).join(', ')
    },
    // Every filename the scene would match plus every matched file, plus names added here
    rows () {
      if (!this.scene) return []
      let listed = []
      try { listed = JSON.parse(this.scene.filenames_arr || '[]') || [] } catch (e) { listed = [] }
      const byName = {}
      for (const f of this.files) byName[f.filename] = f
      const names = Array.from(new Set(listed.concat(Object.keys(byName), this.extraNames))).filter(n => n)
      names.sort(natural)
      return names.map(n => {
        const f = byName[n] || this.unmatched[n]
        const video = f ? f.type === 'video' : !SIDE_EXT.includes(ext(n))
        return { name: n, file: f || null, video: video, pending: !byName[n] && !!f }
      })
    },
    videoRows () {
      return this.rows.filter(r => r.video)
    },
    // Supplemental files: locked to a video (same name with .mp4), attached to the video with the
    // closest name, or loose
    sideInfo () {
      const videos = this.videoRows
      const videoNames = new Set(videos.map(v => v.name))
      let shared = videos.length ? stem(videos[0].name).length : 0
      for (const v of videos) shared = Math.min(shared, commonPrefix(stem(videos[0].name), stem(v.name)))
      const out = {}
      for (const r of this.rows.filter(r => !r.video)) {
        const mp4 = r.name.slice(0, r.name.length - ext(r.name).length) + 'mp4'
        if (SWAP_EXT.includes(ext(r.name)) && videoNames.has(mp4)) {
          out[r.name] = { name: r.name, video: mp4, locked: mp4 }
          continue
        }
        let best = null; let bestLen = -1; let tie = false
        for (const v of videos) {
          const l = commonPrefix(stem(r.name), stem(v.name))
          if (l > bestLen) { best = v.name; bestLen = l; tie = false } else if (l === bestLen) { tie = true }
        }
        if (best && !tie && bestLen > shared) {
          out[r.name] = { name: r.name, video: best, locked: null }
        } else {
          out[r.name] = { name: r.name, video: null, locked: null }
        }
      }
      return out
    },
    groups () {
      return this.videoRows.map(v => ({
        video: v,
        sides: Object.values(this.sideInfo).filter(s => s.video === v.name).sort((a, b) => natural(a.name, b.name))
      }))
    },
    loose () {
      return Object.values(this.sideInfo).filter(s => !s.video).sort((a, b) => natural(a.name, b.name))
    },
    canSplit () {
      if (!this.parts.length) return false
      return this.rows.every(r => this.effective(r.name) !== null && this.effective(r.name) !== undefined)
    }
  },
  methods: {
    imageURL (u) {
      return u.startsWith('http') ? '/img/700x/' + u.replace('://', ':/') : u
    },
    fmtDur (sec) {
      if (!sec) return ''
      const s = Math.round(sec)
      const h = Math.floor(s / 3600); const m = Math.floor((s % 3600) / 60); const r = s % 60
      return (h ? h + ':' + String(m).padStart(2, '0') : m) + ':' + String(r).padStart(2, '0')
    },
    sideIcon (name) {
      return { funscript: 'vibrate', srt: 'file-document-outline', ssa: 'file-document-outline', ass: 'file-document-outline', hsp: 'file-cog-outline' }[ext(name)] || 'file-outline'
    },
    // The "no part number" filler: its name is the start of the other video names (...uhqe.mp4)
    isFiller (row) {
      const s = stem(row.name)
      return this.videoRows.some(v => v.name !== row.name && stem(v.name).startsWith(s))
    },
    resetDefaults () {
      for (let n = 1; n <= this.partCount; n++) {
        if (!(n in this.enabled)) this.$set(this.enabled, n, true)
        if (!this.cards[n]) {
          this.$set(this.cards, n, {
            title: this.scene.title,
            duration: null,
            favourite: !!this.scene.favourite,
            watched: !!this.scene.is_watched,
            hidden: false,
            star_rating: this.scene.star_rating || 0
          })
        }
      }
      const assign = {}
      const videos = this.videoRows
      const matched = videos.filter(v => v.file && !this.isFiller(v))
      const pool = matched.length ? matched : videos.filter(v => !this.isFiller(v))
      let i = 0
      for (const v of videos) {
        if (!pool.includes(v)) {
          assign[v.name] = 0
        } else {
          assign[v.name] = i < this.parts.length ? this.parts[i] : null
          i++
        }
      }
      for (const s of Object.values(this.sideInfo)) {
        assign[s.name] = s.video ? undefined : null
      }
      this.assign = assign
      this.override = {}
    },
    setAssign (name, n) {
      this.$set(this.assign, name, n)
    },
    isFollowing (s) {
      return !!s.locked || (!!s.video && !(s.name in this.override))
    },
    startOverride (s) {
      this.$set(this.override, s.name, true)
      this.$set(this.assign, s.name, this.assign[s.video])
    },
    stopOverride (s) {
      this.$delete(this.override, s.name)
      this.$set(this.assign, s.name, undefined)
    },
    // Where a filename ends up: its own choice, or its video's
    effective (name) {
      const s = this.sideInfo[name]
      if (s && this.isFollowing(s)) return this.assign[s.video]
      return this.assign[name]
    },
    namesIn (n) {
      return this.rows.filter(r => this.effective(r.name) === n).map(r => r.name)
    },
    // Videos and freely assigned files; files that come along through their .mp4 are only counted
    shownNamesIn (n) {
      return this.namesIn(n).filter(f => !(this.sideInfo[f] && this.sideInfo[f].locked))
    },
    lockedIn (n) {
      return this.namesIn(n).filter(f => this.sideInfo[f] && this.sideInfo[f].locked).length
    },
    async lookupUnmatched (names) {
      if (!names.length) return
      try {
        const files = await ky.post('/api/scene/javsplit/unmatched-files', { json: { names: names } }).json()
        for (const f of files) {
          if (!this.unmatched[f.filename]) this.$set(this.unmatched, f.filename, f)
        }
      } catch (e) {}
    },
    videosIn (n) {
      return this.rows.filter(r => r.video && r.file && this.effective(r.name) === n)
    },
    minutes (n) {
      const c = this.cards[n]
      if (c && c.duration !== null && c.duration !== '') return c.duration
      const secs = this.videosIn(n).reduce((t, r) => t + (r.file.duration || 0), 0)
      return Math.round(secs / 60)
    },
    card (n) {
      return this.cards[n] || {}
    },
    addName () {
      const n = this.newName.trim()
      if (!n || this.rows.some(r => r.name === n)) { this.newName = ''; return }
      this.extraNames.push(n)
      this.newName = ''
      this.lookupUnmatched([n])
      this.$nextTick(() => {
        if (this.sideInfo[n]) {
          this.$set(this.assign, n, this.sideInfo[n].video ? undefined : null)
        } else {
          this.$set(this.assign, n, null)
        }
      })
    },
    async doSplit () {
      const payload = {
        scene_id: this.scene.id,
        original: this.original,
        keep: this.rows.filter(r => this.effective(r.name) === 0).map(r => r.name),
        parts: this.parts.map(n => ({
          number: n,
          title: this.cards[n].title,
          duration: parseInt(this.minutes(n)) || 0,
          filenames: this.namesIn(n),
          favourite: this.cards[n].favourite,
          watched: this.cards[n].watched,
          hidden: this.cards[n].hidden,
          star_rating: this.cards[n].star_rating
        }))
      }
      this.busy = true
      try {
        const res = await ky.post('/api/scene/javsplit', { json: payload, timeout: false }).json()
        this.$buefy.toast.open({ message: `Split ${this.scene.scene_id} into ${res.scenes.length} scenes`, type: 'is-success' })
        this.$emit('close')
        this.$store.dispatch('sceneList/load', { offset: 0 })
        this.$store.commit('overlay/hideDetails')
      } catch (e) {
        let msg = 'Split failed'
        try { msg = (await e.response.json()).message || msg } catch (_) {}
        this.$buefy.toast.open({ message: msg, type: 'is-danger', duration: 8000 })
      }
      this.busy = false
    }
  }
}
</script>

<style>
.split-scene-modal .modal-content, .split-scene-modal .animation-content {
  width: min(1100px, 96vw) !important;
  max-width: none !important;
}
</style>

<style scoped>
.split-card {
  width: 100%;
  height: 94vh;
  max-height: 94vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.split-body {
  flex: 1;
  min-height: 0;
  /* safety net for very short windows; normally only the two panes scroll */
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  position: relative;
}
.split-green, .split-card >>> .split-green {
  color: #2dc560;
}
.split-mhead {
  padding: 8px 16px;
}
.split-mhead .modal-card-title {
  font-size: 1.1rem;
  line-height: 1.2;
  margin: 0 !important;
}
.split-body {
  padding: 10px 16px;
}
.split-head {
  display: flex;
  gap: 14px;
  margin-bottom: 4px;
  line-height: 1.3;
}
/* the details overlay is inside .content, whose p and label margins would pad everything out */
.split-head p, .split-scard p, .split-label {
  margin-bottom: 0 !important;
}
.split-title {
  margin: 0 0 2px;
}
.split-cover {
  width: 110px;
  height: auto;
  align-self: flex-start;
  border-radius: 4px;
}
.split-parts-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
  margin: 4px 0 2px;
}
.split-parts-row .label {
  margin: 0 6px 0 0;
}
.split-pill {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  border: 1px solid #dbdbdb;
  border-radius: 4px;
  padding: 1px 8px;
  font-size: 0.85em;
}
.split-pill.on {
  border-color: #2dc560;
  background: #e6f8ec;
}
.split-warn {
  margin-bottom: 4px !important;
  font-size: 0.72rem;
}
.split-warn .message-body {
  padding: 6px 12px;
}
.split-warn ul {
  margin: 0 0 0 1.2em;
  list-style: disc;
}
.split-label {
  font-size: 0.75em;
  color: #7a7a7a;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  margin: 6px 0 3px;
}
.split-hint {
  text-transform: none;
  letter-spacing: 0;
  color: #b5b5b5;
  margin-left: 6px;
}
/* Both panes scroll on their own, with a scrollbar that is always there */
.split-pane {
  border: 1px solid #dbdbdb;
  border-radius: 4px;
  scrollbar-width: auto;
}
.split-pane::-webkit-scrollbar {
  width: 12px;
  height: 12px;
}
.split-pane::-webkit-scrollbar-track {
  background: #f0f0f0;
}
.split-pane::-webkit-scrollbar-thumb {
  background: #b5b5b5;
  border-radius: 6px;
  border: 2px solid #f0f0f0;
}
.split-files {
  flex: 1;
  min-height: 150px;
  overflow-y: scroll;
}
.split-files thead th {
  position: sticky;
  top: 0;
  background: #fff;
  z-index: 2;
}
.split-files table {
  margin-bottom: 0;
  font-size: 0.8rem;
}
.split-files td, .split-files th {
  padding: 0.2em 0.5em;
}
.split-files .tag {
  font-size: 0.7rem;
  height: 1.7em;
}
.split-col {
  text-align: center !important;
  width: 70px;
}
.split-col input {
  accent-color: #2dc560;
}
.split-col.follow input {
  opacity: 0.45;
}
.split-video td {
  font-weight: 600;
  border-top: 2px solid #dbdbdb;
}
.split-nameonly td {
  font-weight: 400;
  color: #7a7a7a;
}
.split-inset {
  padding-left: 2.4em !important;
}
.split-dur {
  white-space: nowrap;
  width: 90px;
}
.split-loosehead td {
  border-top: 2px solid #dbdbdb;
  font-size: 0.8em;
  color: #7a7a7a;
  text-transform: uppercase;
}
.split-cards {
  flex: none;
  overflow-x: scroll;
  overflow-y: hidden;
  display: flex;
  gap: 10px;
  padding: 8px;
}
.split-scard {
  flex: 0 0 440px;
  border: 1px solid #dbdbdb;
  border-top: 4px solid #2dc560;
  border-radius: 4px;
  padding: 6px 10px 8px;
}
.split-scard .label {
  margin: 4px 0 1px !important;
}
.split-scard .help {
  margin-top: 1px;
}
/* id + tag, stars, then the three toggles in the top right corner */
.split-trow {
  display: flex;
  align-items: center;
  gap: 10px;
}
.split-cid {
  margin: 0;
  white-space: nowrap;
}
.split-trow .vue-star-rating {
  margin-left: auto;
}
.split-trow .buttons {
  margin: 0;
}
.split-trow .button {
  margin-bottom: 0 !important;
}
.split-crow {
  display: flex;
  gap: 14px;
}
.split-will code {
  display: block;
  font-size: 0.75em;
  margin-bottom: 2px;
  word-break: break-all;
}
.split-tag {
  background: #e6f8ec;
  color: #1d7f3d;
}
.split-foot {
  justify-content: space-between;
}
.split-orig {
  margin-left: 14px;
}
.split-orig input {
  accent-color: #2dc560;
}
.split-permanent {
  margin-left: 6px;
  font-size: 0.8em;
  color: #f14668;
}
.split-go {
  background: #2dc560;
  border-color: #2dc560;
  color: #fff;
}
.split-go:hover {
  background: #26a852;
  color: #fff;
}
</style>
