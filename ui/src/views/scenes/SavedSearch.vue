<template>
  <div>
    <b-field grouped>
      <b-select size="is-small" @input="setPlaylist" expanded v-model="currentPlaylist">
        <optgroup label="Web">
          <option v-for="(obj, idx) in playlistsWeb" :value="obj.id" :key="idx">
            {{ obj.name }}
          </option>
        </optgroup>
        <optgroup label="VR Players">
          <option v-for="(obj, idx) in playlistsDeo" :value="obj.id" :key="idx">
            {{ obj.name }}
          </option>
        </optgroup>
      </b-select>

      <b-tooltip position="is-bottom" label="Save as new" :delay="200">
        <button class="button is-small is-outlined" @click="showNewDialog">
          <b-icon pack="mdi" icon="content-save-outline"></b-icon>
        </button>
      </b-tooltip>
      <b-tooltip position="is-bottom" label="Edit" :delay="200">
        <button class="button is-small is-outlined" @click="showEditDialog" :disabled="disableEditDelete">
          <b-icon pack="mdi" icon="square-edit-outline"></b-icon>
        </button>
      </b-tooltip>
      <b-tooltip position="is-bottom" label="Delete" :delay="200">
        <button class="button is-small is-outlined" @click="removePlaylist" :disabled="disableEditDelete">
          <b-icon pack="mdi" icon="delete-outline"></b-icon>
        </button>
      </b-tooltip>
    </b-field>

    <b-modal :active.sync="isPlaylistModalActive"
             has-modal-card
             trap-focus
             aria-role="dialog"
             aria-modal>
      <div class="modal-card" style="width: auto">
        <header class="modal-card-head">
          <p class="modal-card-title">{{ modalTitle }}</p>
        </header>
        <section class="modal-card-body">
          <b-field label="Name">
            <b-input
              type="name"
              v-model="playlistName"
              required>
            </b-input>
          </b-field>
          <b-checkbox v-model="playlistDeoEnabled">Use as DeoVR list</b-checkbox>
        </section>
        <footer class="modal-card-foot">
          <button class="button is-primary" :disabled="playlistName===''" @click="savePlaylist(modalAction)">Save
          </button>
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
      currentPlaylistObj: null,

      isPlaylistModalActive: false,
      modalTitle: '',
      modalAction: 'create',
      playlistName: '',
      playlistDeoEnabled: false
    }
  },
  methods: {
    showNewDialog () {
      this.modalTitle = 'Create new saved search'
      this.modalAction = 'create'
      this.playlistName = ''
      this.playlistDeoEnabled = false

      this.isPlaylistModalActive = true
    },
    showEditDialog () {
      if (this.currentPlaylistObj !== null) {
        this.modalTitle = 'Edit saved search'
        this.modalAction = 'update'
        this.playlistName = this.currentPlaylistObj.name
        this.playlistDeoEnabled = this.currentPlaylistObj.is_deo_enabled

        this.isPlaylistModalActive = true
      }
    },
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
    async savePlaylist (action) {
      const payload = {
        name: this.playlistName,
        is_deo_enabled: this.playlistDeoEnabled,
        is_smart: true,
        search_params: JSON.stringify(this.$store.state.sceneList.filters)
      }

      let p
      if (action === 'create') {
        p = await ky.post('/api/playlist', { json: payload }).json()
      } else {
        p = await ky.put(`/api/playlist/${this.currentPlaylistObj.id}`, { json: payload }).json()
      }

      await this.$store.dispatch('sceneList/filters')
      this.currentPlaylist = p.id
      this.isPlaylistModalActive = false
    },
    removePlaylist () {
      this.$buefy.dialog.confirm({
        title: 'Delete saved search',
        message: `Do you want to delete saved search <strong>${this.currentPlaylistObj.name}</strong>?`,
        type: 'is-danger',
        hasIcon: true,
        confirmText: 'Delete',
        onConfirm: () => {
          ky.delete(`/api/playlist/${this.currentPlaylistObj.id}`).then(() => {
            this.$store.dispatch('sceneList/filters')
            this.currentPlaylist = null
          })
        }
      })
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
    },
    currentPlaylist: {
      get () {
        if (this.currentPlaylistObj !== null) {
          return this.currentPlaylistObj.id
        }
        return null
      },
      set (val) {
        if (val === null) {
          this.currentPlaylistObj = null
          return null
        }
        this.currentPlaylistObj = this.playlists.find(item => item.id === val)
        return this.currentPlaylistObj.id
      }
    },
    disableEditDelete () {
      if (this.currentPlaylistObj === null || this.currentPlaylistObj.is_system === true) {
        return true
      }
      return false
    }
  }
}
</script>

<style lang="scss" scoped>
button {
  margin-left: 0.1rem;
}
</style>
