<template>
  <div class="container">
    <b-loading :is-full-page="false" :active.sync="isLoading" />
    <div class="content">
      <h3>{{ $t('Organize files') }}</h3>
      <p>
        Organize your videos into folders <code>{{ layoutPreview }}</code>.
      </p>
      <hr />

      <b-field><b-switch v-model="dedup" type="is-danger">Delete byte-identical duplicate copies (md5-confirmed)</b-switch></b-field>
      <b-field><b-switch v-model="deferDups" type="is-default">Defer scenes with possible duplicates (skip the md5 pass)</b-switch></b-field>
      <b-field label="Staging subfolder (relative to each storage folder; files held until old enough)">
        <b-input v-model="incomingDir" placeholder="Incoming" />
      </b-field>
      <b-field label="Staging grace period (days)">
        <b-numberinput v-model="incomingMinAge" :min="0" :max="365" :step="1" controls-position="compact" />
      </b-field>
      <b-field label="Top folder (wrapper under the volume root; blank = studio folders in the root)">
        <b-input v-model="topFolder" placeholder="(none)" />
      </b-field>
      <b-field label="Cast in folder name">
        <b-select v-model="castGender" expanded>
          <option value="any">No preference (all performers)</option>
          <option value="female">Woman</option>
          <option value="male">Man</option>
        </b-select>
      </b-field>
      <b-field><b-switch v-model="symlinkByActor" type="is-default">Also symlink each scene into a per-actor folder (needs a filesystem that supports symlinks)</b-switch></b-field>
      <b-field v-if="symlinkByActor" label="Actor folder (parent for the CamelCase per-actor folders)">
        <b-input v-model="actorFolder" placeholder="ByActor" />
      </b-field>
      <b-field label="Limit (0 = all; consider at most N video files)">
        <b-numberinput v-model="limit" :min="0" :step="10" controls-position="compact" />
      </b-field>
      <div class="buttons">
        <b-button type="is-primary" @click="save">Save settings</b-button>
        <b-button type="is-link" :disabled="running" @click="preview">Preview (dry-run)</b-button>
        <b-button type="is-danger" :disabled="running || !hasPreview" @click="apply">Apply</b-button>
      </div>

      <b-message v-if="running" type="is-warning">A run is in progress…</b-message>
      <div v-if="result">
        <hr />
        <h4>{{ result.dryRun ? 'Preview' : 'Applied' }}</h4>
        <table class="table is-narrow is-fullwidth">
          <tbody>
            <tr><td>Scenes acted</td><td>{{ result.scenesActed }}</td></tr>
            <tr><td>Files moved</td><td>{{ result.filesMoved }}</td></tr>
            <tr><td>Files renamed</td><td>{{ result.filesRenamed }}</td></tr>
            <tr><td>Identical copies deleted</td><td>{{ result.identicalCopiesDeleted }} (~{{ (result.bytesReclaimed/1e9).toFixed(1) }} GB)</td></tr>
            <tr><td>Hard links unlinked</td><td>{{ result.hardlinksUnlinked }}</td></tr>
            <tr><td>Sidecars moved</td><td>{{ result.sidecarsMoved }}</td></tr>
            <tr><td>Actor symlinks created</td><td>{{ result.actorSymlinksCreated }}</td></tr>
            <tr><td>Actor symlinks pruned</td><td>{{ result.actorSymlinksPruned }}</td></tr>
            <tr><td>Empty dirs removed</td><td>{{ result.emptyDirsRemoved }}</td></tr>
            <tr><td>Scenes deferred</td><td>{{ result.scenesDeferred }}</td></tr>
            <tr><td>Files held (staging)</td><td>{{ result.filesHeldRecent }}</td></tr>
            <tr><td>Merged duplicate scenes</td><td>{{ result.mergedDuplicateScenes }}</td></tr>
          </tbody>
        </table>
        <h5>First actions</h5>
        <pre style="max-height:20em;overflow:auto">{{ sampleActions }}</pre>
      </div>
    </div>
  </div>
</template>

<script>
const FIELDS = ['dedup', 'deferDups', 'incomingDir', 'incomingMinAge', 'topFolder', 'castGender', 'symlinkByActor', 'actorFolder']

export default {
  name: 'Organize',
  data () {
    return { limit: 100, poll: null }
  },
  computed: {
    isLoading () { return this.$store.state.optionsOrganize.loading },
    running () { return this.$store.state.optionsOrganize.running },
    result () { return this.$store.state.optionsOrganize.result },
    hasPreview () { return this.result && this.result.dryRun },
    layoutPreview () {
      return (this.topFolder ? this.topFolder + '/' : '') + '{Studio}/{Studio}.{YY.MM.DD}.{Cast}.{Title}.XXX.{FOV}.{Height}p/'
    },
    sampleActions () {
      if (!this.result || !this.result.actions) return '(none)'
      return this.result.actions.slice(0, 200)
        .map(a => `${a.kind}\t${a.from}${a.to ? '  ->  ' + a.to : ''}${a.note ? '  ' + a.note : ''}`).join('\n')
    },
    ...FIELDS.reduce((acc, key) => {
      acc[key] = {
        get () { return this.$store.state.optionsOrganize.config[key] },
        set (value) { this.$store.commit('optionsOrganize/setField', { key, value }) }
      }
      return acc
    }, {})
  },
  methods: {
    save () {
      this.$store.dispatch('optionsOrganize/save').then(() => {
        this.$buefy.toast.open({ message: 'Organize settings saved', type: 'is-success' })
      })
    },
    startPolling () {
      if (this.poll) return
      this.poll = setInterval(async () => {
        const st = await this.$store.dispatch('optionsOrganize/pollStatus')
        if (!st.running) { clearInterval(this.poll); this.poll = null }
      }, 2000)
    },
    async preview () {
      await this.$store.dispatch('optionsOrganize/save')
      await this.$store.dispatch('optionsOrganize/run', { dryRun: true, limit: this.limit })
      this.$buefy.toast.open({ message: 'Preview running…', type: 'is-info' })
      this.startPolling()
    },
    apply () {
      this.$buefy.dialog.confirm({
        title: 'Apply reorganisation',
        message: 'This will move/rename/delete files on disk. Proceed?',
        type: 'is-danger',
        confirmText: 'Apply',
        onConfirm: async () => {
          await this.$store.dispatch('optionsOrganize/run', { dryRun: false, limit: this.limit })
          this.$buefy.toast.open({ message: 'Applying…', type: 'is-warning' })
          this.startPolling()
        }
      })
    }
  },
  mounted () { this.$store.dispatch('optionsOrganize/load') },
  beforeDestroy () { if (this.poll) clearInterval(this.poll) }
}
</script>
