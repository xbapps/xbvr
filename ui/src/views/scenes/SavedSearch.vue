<template>
  <div>
    <b-select size="is-small" @input="setPlaylist">
      <optgroup label="Web">
        <option v-for="(obj, idx) in playlistsWeb" :value="obj.id" :key="obj.id">
          {{ obj.name }}
        </option>
      </optgroup>
      <optgroup label="VR Players">
        <option v-for="(obj, idx) in playlistsDeo" :value="obj.id" :key="obj.id">
          {{ obj.name }}
        </option>
      </optgroup>
    </b-select>

    <b-button size="is-small" @click="isPlaylistModalActive=true">Save as smart playlist</b-button>

    <b-modal :active.sync="isPlaylistModalActive"
             has-modal-card
             trap-focus
             aria-role="dialog"
             aria-modal>
      <div class="modal-card" style="width: auto">
        <header class="modal-card-head">
          <p class="modal-card-title">Create new smart playlist</p>
        </header>
        <section class="modal-card-body">
          <b-field label="Playlist name">
            <b-input
              type="name"
              v-model="playlistName"
              placeholder="Give playlist a name"
              required>
            </b-input>
          </b-field>
          <b-checkbox v-model="playlistDeoEnabled">Use as DeoVR list</b-checkbox>
        </section>
        <footer class="modal-card-foot">
          <button class="button is-primary" :disabled="playlistName===''" @click="savePlaylist">Save</button>
        </footer>
      </div>
    </b-modal>
  </div>
</template>

<script>
import ky from 'ky'

export default {
  name: 'SavedSearch',
  mounted () {
    this.$store.dispatch('sceneList/filters')
  },
  data () {
    return {
      playlistName: '',
      playlistDeoEnabled: false,
      isPlaylistModalActive: false
    }
  },
  methods: {
    setPlaylist (val) {
      const obj = this.playlists.find(item => item.id === val)
      this.$router.push({
        name: 'scenes',
        query: {
          q: this.$store.getters['sceneList/getQueryParamsFromObject'](obj.search_params)
        }
      })

      this.$store.dispatch('sceneList/load', { offset: 0 })
    },
    async savePlaylist () {
      await ky.post('/api/playlist', {
        json: {
          name: this.playlistName,
          is_deo_enabled: this.playlistDeoEnabled,
          is_smart: true,
          search_params: JSON.stringify(this.$store.state.sceneList.filters)
        }
      })
      this.isPlaylistModalActive = false
      await this.$store.dispatch('sceneList/filters')
    }
  },
  computed: {
    playlists () {
      return this.$store.state.sceneList.playlists
    },
    playlistsWeb () {
      return this.$store.state.sceneList.playlists.filter((obj) => {
        return obj.is_deo_enabled === false
      })
    },
    playlistsDeo () {
      return this.$store.state.sceneList.playlists.filter((obj) => {
        return obj.is_deo_enabled === true
      })
    }
  }
}
</script>
