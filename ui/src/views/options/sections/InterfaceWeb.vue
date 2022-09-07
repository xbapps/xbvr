<template>
  <div class="container">
    <b-loading :is-full-page="false" :active.sync="isLoading" />
    <div class="content">
      <h3>{{ $t('Web UI') }}</h3>
      <hr />
      <div class="columns">
        <div class="column">
          <section>
            <b-field label="Tag Sort">
              <div class="block">
                <b-radio v-model="tagSort" name="tagSort" native-value="by-tag-count">
                  By Tag Count
                </b-radio>
                <b-radio v-model="tagSort" name="tagSort" native-value="alphabetically">
                  Alphabetically
                </b-radio>
              </div>
            </b-field>

            <b-field label="Buttons in Scene List">
            </b-field>
            <b-field>
              <b-switch v-model="sceneWatchlist" type="is-default">
                show Add/Remove from Watchlist button
              </b-switch>
            </b-field>
            <b-field>
              <b-switch v-model="sceneFavourite" type="is-danger">
                show Add/Remove from Favourites button
              </b-switch>
            </b-field>
            <b-field>
              <b-switch v-model="sceneWatched" type="is-dark">
                show Toggle Watched Status button
              </b-switch>
            </b-field>
            <b-field>
              <b-switch v-model="sceneEdit" type="is-dark">
                show Edit Scene button
              </b-switch>
            </b-field>
            <b-field>
              <b-switch v-model="sceneCuepoint" type="is-dark">
                show Cuepoints  button
              </b-switch>
            </b-field>
            <b-field>
              <b-switch v-model="hspFile" type="is-dark">
                show Hsp File button
              </b-switch>
            </b-field>

            <b-field label="Automatically Check for Updates">
              <b-switch v-model="updateCheck">
                Enabled
              </b-switch>
            </b-field>

            <b-field>
              <b-button type="is-primary" @click="save">Save</b-button>
            </b-field>
          </section>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  name: 'InterfaceWeb',
  mounted () {
    this.$store.dispatch('optionsWeb/load')
  },
  methods: {
    save () {
      this.$store.dispatch('optionsWeb/save')
    }
  },
  computed: {
    tagSort: {
      get () {
        return this.$store.state.optionsWeb.web.tagSort
      },
      set (value) {
        this.$store.state.optionsWeb.web.tagSort = value
      }
    },
    sceneWatchlist: {
      get () {
        return this.$store.state.optionsWeb.web.sceneWatchlist
      },
      set (value) {
        this.$store.state.optionsWeb.web.sceneWatchlist = value
      }
    },
    sceneFavourite: {
      get () {
        return this.$store.state.optionsWeb.web.sceneFavourite
      },
      set (value) {
        this.$store.state.optionsWeb.web.sceneFavourite = value
      }
    },
    sceneWatched: {
      get () {
        return this.$store.state.optionsWeb.web.sceneWatched
      },
      set (value) {
        this.$store.state.optionsWeb.web.sceneWatched = value
      }
    },
    sceneEdit: {
      get () {
        return this.$store.state.optionsWeb.web.sceneEdit
      },
      set (value) {
        this.$store.state.optionsWeb.web.sceneEdit = value
      }
    },
    updateCheck: {
      get () {
        return this.$store.state.optionsWeb.web.updateCheck
      },
      set (value) {
        this.$store.state.optionsWeb.web.updateCheck = value
      }
    },
    sceneCuepoint: {
      get () {
        return this.$store.state.optionsWeb.web.sceneCuepoint
      },
      set (value) {
        this.$store.state.optionsWeb.web.sceneCuepoint = value
      }
    },
    hspFile: {
      get () {
        return this.$store.state.optionsWeb.web.showHspFile
      },
      set (value) {
        this.$store.state.optionsWeb.web.showHspFile = value
      }
    },
    isLoading: function () {
      return this.$store.state.optionsWeb.loading
    }
  }
}
</script>

<style scoped>

</style>
