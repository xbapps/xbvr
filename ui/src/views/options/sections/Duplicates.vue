<template>
  <div class="container">
    <b-loading :is-full-page="false" :active.sync="loading" />
    <div class="content">
      <h3>{{ $t('Duplicate files') }}</h3>
      <p>
        Scenes with more than one distinct video file. Files of clearly different length are flagged
        as a possible <strong>mis-assignment</strong>; otherwise their <strong>PSNR</strong> is compared
        to suggest which to keep. Tick the files you want to remove, or ignore a scene forever.
      </p>
      <div class="buttons is-align-items-center">
        <b-button type="is-link" :disabled="running" @click="analyze(false)">Analyze</b-button>
        <b-button :disabled="running" @click="analyze(true)">Re-analyze all</b-button>
        <b-button type="is-danger" :disabled="checkedCount === 0" @click="deleteTicked">Delete ticked ({{ checkedCount }})</b-button>
        <b-button type="is-warning" :disabled="checkedCount === 0" @click="disassociateTicked">Disassociate ticked ({{ checkedCount }})</b-button>
        <b-checkbox v-model="showIgnored" class="ml-3">Show ignored</b-checkbox>
        <span v-if="running" class="ml-3">Analyzing… {{ done }}/{{ total }}</span>
        <span v-else class="ml-3">{{ groups.length }} group(s)</span>
      </div>
      <hr />

      <div v-for="g in groups" :key="g.sceneId" class="box">
        <div>
          <strong>{{ g.site }}</strong> — {{ g.title }}
          <b-tag :type="statusType(g.status)" class="ml-2">{{ g.status }}</b-tag>
        </div>
        <p class="is-size-7 has-text-grey">{{ g.detail }}</p>
        <table class="table is-narrow is-fullwidth is-size-7">
          <thead>
            <tr><th>del</th><th>file</th><th>res</th><th>bitrate</th><th>duration</th><th>size</th><th>PSNR vs best</th></tr>
          </thead>
          <tbody>
            <tr v-for="f in g.files" :key="f.fileId"
                :class="{ 'has-background-success-light': f.fileId === g.keepFileId && !f.ignored, 'has-text-grey-light': f.ignored }">
              <td><b-checkbox v-model="checked[f.fileId]" /></td>
              <td>
                {{ f.filename }}
                <b-tag v-if="f.ignored" type="is-light" size="is-small">ignored</b-tag>
                <b-tag v-else-if="f.fileId === g.keepFileId" type="is-success" size="is-small">suggested keep</b-tag>
                <b-tag v-else-if="f.suggest === 'review'" type="is-warning" size="is-small">review</b-tag>
                <b-button size="is-small" type="is-text" icon-left="play" @click="review(f)">Review</b-button>
                <b-button size="is-small" type="is-text" @click="toggleIgnore(f)">{{ f.ignored ? 'Unignore' : 'Ignore' }}</b-button>
              </td>
              <td>{{ f.height }}p</td>
              <td>{{ (f.bitrate/1e6).toFixed(0) }}M</td>
              <td>{{ fmtDur(f.duration) }}</td>
              <td>{{ (f.size/1e9).toFixed(1) }}G</td>
              <td>{{ f.fileId === g.keepFileId ? '—' : (f.psnr >= 99 ? 'identical' : f.psnr) }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
    <Player v-if="showPlayerOverlay" />
  </div>
</template>

<script>
import Player from '../../files/Player.vue'

export default {
  name: 'Duplicates',
  components: { Player },
  data () { return { checked: {}, poll: null } },
  computed: {
    loading () { return this.$store.state.optionsDuplicates.loading },
    running () { return this.$store.state.optionsDuplicates.running },
    done () { return this.$store.state.optionsDuplicates.done },
    total () { return this.$store.state.optionsDuplicates.total },
    groups () { return this.$store.state.optionsDuplicates.groups },
    showPlayerOverlay () { return this.$store.state.overlay.player.show },
    showIgnored: {
      get () { return this.$store.state.optionsDuplicates.showIgnored },
      set (v) {
        this.$store.commit('optionsDuplicates/setShowIgnored', v)
        this.$store.dispatch('optionsDuplicates/load')
      }
    },
    checkedCount () { return Object.values(this.checked).filter(Boolean).length }
  },
  methods: {
    review (f) {
      this.$store.commit('overlay/showPlayer', { file: { id: f.fileId, projection: f.projection } })
    },
    toggleIgnore (f) {
      const action = f.ignored ? 'optionsDuplicates/unignore' : 'optionsDuplicates/ignore'
      this.$store.dispatch(action, f.fileId).then(() => this.$store.dispatch('optionsDuplicates/load'))
    },
    statusType (s) {
      return { 'duration-mismatch': 'is-danger', 'low-psnr': 'is-warning', 'identical': 'is-success', 'same-content': 'is-info' }[s] || 'is-light'
    },
    fmtDur (sec) { return Math.floor(sec / 60) + 'm' + Math.round(sec % 60) + 's' },
    startPoll () {
      if (this.poll) return
      this.poll = setInterval(async () => {
        await this.$store.dispatch('optionsDuplicates/load')
        if (!this.running) { clearInterval(this.poll); this.poll = null }
      }, 3000)
    },
    async analyze (force) {
      await this.$store.dispatch('optionsDuplicates/analyze', force)
      this.$buefy.toast.open({ message: 'Analyzing duplicates…', type: 'is-info' })
      this.startPoll()
    },
    deleteTicked () {
      const ids = Object.keys(this.checked).filter(id => this.checked[id]).map(Number)
      if (!ids.length) return
      this.$buefy.dialog.confirm({
        title: 'Delete files',
        message: `Delete ${ids.length} ticked file(s) from disk (across all scenes)?`,
        type: 'is-danger',
        confirmText: 'Delete',
        onConfirm: async () => {
          await this.$store.dispatch('optionsDuplicates/del', ids)
          this.checked = {}
          await this.$store.dispatch('optionsDuplicates/load')
          this.$buefy.toast.open({ message: `Deleted ${ids.length} file(s)`, type: 'is-success' })
        }
      })
    },
    disassociateTicked () {
      const ids = Object.keys(this.checked).filter(id => this.checked[id]).map(Number)
      if (!ids.length) return
      this.$buefy.dialog.confirm({
        title: 'Disassociate files',
        message: `Detach ${ids.length} ticked file(s) from their scene? The files stay on disk but are unmatched (won't be auto-rematched).`,
        confirmText: 'Disassociate',
        onConfirm: async () => {
          await this.$store.dispatch('optionsDuplicates/disassociate', ids)
          this.checked = {}
          await this.$store.dispatch('optionsDuplicates/load')
          this.$buefy.toast.open({ message: `Disassociated ${ids.length} file(s)`, type: 'is-success' })
        }
      })
    }
  },
  mounted () { this.$store.dispatch('optionsDuplicates/load') },
  beforeDestroy () { if (this.poll) clearInterval(this.poll) }
}
</script>
