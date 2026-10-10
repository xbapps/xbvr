<template>
  <span v-if="visible">
    <a class="button is-small split-button" @click="open = true" title="Split scene into parts">
      <b-icon pack="mdi" icon="call-split" size="is-small"/>
    </a>
    <b-modal :active.sync="open" has-modal-card trap-focus :width="1100" :can-cancel="['escape', 'x']" custom-class="split-scene-modal" aria-role="dialog" aria-modal>
      <split-scene :item="item" @close="open = false"/>
    </b-modal>
  </span>
</template>

<script>
import SplitScene from './SplitScene'

export default {
  name: 'SplitButton',
  components: { SplitScene },
  props: { item: Object },
  data () {
    return { open: false }
  },
  computed: {
    // Only JAV scenes from the R18.dev scrapers or the JAV import box, never a split scene
    visible () {
      const s = this.item
      if (!s) return false
      if (/_split\d+$/.test(s.scene_id || '')) return false
      if ((s.tags || []).some(t => t.name === 'split javr')) return false
      if ((s.scraper_id || '').startsWith('r18dev-')) return true
      const u = (s.scene_url || '').toLowerCase()
      return !s.scraper_id && (u.includes('r18.dev') || u.includes('dmm.co.jp'))
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
</style>
