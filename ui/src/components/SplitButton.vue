<template>
  <span v-if="javr && status">
    <a v-if="status.can_split" class="button is-small split-button" @click="open = true" title="Split scene into parts">
      <b-icon pack="mdi" icon="call-split" size="is-small"/>
    </a>
    <a v-if="status.can_undo" class="button is-small merge-button" @click="undo" :title="'Merge the splits of ' + status.original_scene_id + ' back together'">
      <b-icon pack="mdi" icon="call-merge" size="is-small"/>
    </a>
    <b-modal v-if="status.can_split" :active.sync="open" has-modal-card trap-focus :width="1100" :can-cancel="['escape', 'x']" custom-class="split-scene-modal" aria-role="dialog" aria-modal>
      <split-scene :item="item" @close="open = false"/>
    </b-modal>
  </span>
</template>

<script>
import ky from 'ky'
import SplitScene from './SplitScene'

export default {
  name: 'SplitButton',
  components: { SplitScene },
  props: { item: Object },
  data () {
    return { open: false, status: null }
  },
  computed: {
    // Only JAV scenes from the R18.dev scrapers or the JAV import box (split scenes are too)
    javr () {
      const s = this.item
      if (!s) return false
      if ((s.scraper_id || '').startsWith('r18dev-')) return true
      const u = (s.scene_url || '').toLowerCase()
      return !s.scraper_id && (u.includes('r18.dev') || u.includes('dmm.co.jp'))
    }
  },
  watch: {
    'item.id': {
      immediate: true,
      handler () { this.loadStatus() }
    }
  },
  methods: {
    // Split, undo or nothing, depending on whether the scene is split / has splits / was deleted
    async loadStatus () {
      this.status = null
      if (!this.javr) return
      const id = this.item.id
      try {
        const st = await ky.get(`/api/scene/javsplit/status/${id}`).json()
        if (this.item && this.item.id === id) this.status = st
      } catch (e) {}
    },
    undo () {
      const st = this.status
      this.$buefy.dialog.confirm({
        title: 'Merge splits back together',
        message: `Put <strong>${st.original_scene_id}</strong> back together?<br><br>` +
          `${st.splits.join(', ')} get deleted. Their files and filenames go back to ${st.original_scene_id}, ` +
          `which is shown again. Edits made to the split scenes are lost.`,
        confirmText: 'Merge',
        type: 'is-warning',
        hasIcon: true,
        onConfirm: async () => {
          try {
            await ky.post('/api/scene/javsplit/undo', { json: { scene_id: this.item.id }, timeout: false }).json()
            this.$buefy.toast.open({ message: `${st.original_scene_id} is back in one piece`, type: 'is-success' })
            this.$store.dispatch('sceneList/load', { offset: 0 })
            this.$store.commit('overlay/hideDetails')
          } catch (e) {
            let msg = 'Undo failed'
            try { msg = (await e.response.json()).message || msg } catch (_) {}
            this.$buefy.toast.open({ message: msg, type: 'is-danger', duration: 8000 })
          }
        }
      })
    }
  }
}
</script>

<style scoped>
.split-button {
  border-color: #2dc560;
  color: #2dc560;
  background-color: transparent;
}
.split-button:hover {
  background-color: #2dc560;
  color: #fff;
}
/* Merge (undo split): the yellow that goes with the split green. #ffff99 is too light for an
   outline on white, so it's the fill, with a dark icon. */
.merge-button {
  background-color: #ffff99;
  border-color: #e0d84a;
  color: #363636;
}
.merge-button:hover {
  background-color: #f5f06e;
  border-color: #c9c03a;
  color: #363636;
}
</style>
