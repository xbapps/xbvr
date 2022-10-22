<template>
  <div>
    <div class="is-divider" data-content="Saved searches" style="margin-top:0.8em;"></div>

    <SavedSearch/>

    <div class="is-divider" data-content="Properties"></div>

    <div class="columns is-multiline is-gapless">
      <div class="column is-half">
        <b-checkbox-button v-model="lists" native-value="watchlist" type="is-primary">
          <b-icon pack="mdi" icon="calendar-check"/>
          <span>{{ $t('Watchlist') }}</span>
        </b-checkbox-button>
      </div>
      <div class="column is-half">
        <b-checkbox-button v-model="lists" native-value="favourite" type="is-danger">
          <b-icon pack="mdi" icon="heart"/>
          <span>{{ $t('Favourite') }}</span>
        </b-checkbox-button>
      </div>
      <div class="column is-half">
        <b-checkbox-button v-model="lists" native-value="scripted" type="is-info">
          <b-icon pack="mdi" icon="pulse"/>
          <span>{{ $t('Scripted') }}</span>
        </b-checkbox-button>
      </div>
    </div>

    <div class="is-divider" data-content="Sorting / Status / Release"></div>

    <b-field :label="$t('Sort by')" label-position="on-border" :addons="true" class="field-extra">
      <div class="control is-expanded">
        <div class="select is-fullwidth">
          <select v-model="sort">
            <option value="release_desc">↓ {{ $t("Release date") }}</option>
            <option value="release_asc">↑ {{ $t("Release date") }}</option>
            <option value="added_desc">↓ {{ $t("File added date") }}</option>
            <option value="added_asc">↑ {{ $t("File added date") }}</option>
            <option value="total_file_size_desc">↓ {{ $t("File size") }}</option>
            <option value="total_file_size_asc">↑ {{ $t("File size") }}</option>
            <option value="rating_desc">↓ {{ $t("Rating") }}</option>
            <option value="rating_asc">↑ {{ $t("Rating") }}</option>
            <option value="total_watch_time_desc">↓ {{ $t("Watch time") }}</option>
            <option value="total_watch_time_asc">↑ {{ $t("Watch time") }}</option>
            <option value="scene_added_desc">↓ {{ $t("Scene added date") }}</option>
            <option value="scene_updated_desc">↓ {{ $t("Scene updated date") }}</option>
            <option value="last_opened_desc">↓ {{ $t("Last viewed date") }}</option>
            <option value="last_opened_asc">↑ {{ $t("Last viewed date") }}</option>
            <option value="random">↯ {{ $t("Random") }}</option>
          </select>
        </div>
      </div>
    </b-field>

    <b-field label="Watched" label-position="on-border" :addons="true" class="field-extra">
      <div class="control is-expanded">
        <div class="select is-fullwidth">
          <select v-model="isWatched">
            <option :value="null">Everything</option>
            <option :value="true">Watched</option>
            <option :value="false">Unwatched</option>
          </select>
        </div>
      </div>
    </b-field>

    <b-field label="Release month" label-position="on-border" :addons="true" class="field-extra">
      <div class="control is-expanded">
        <div class="select is-fullwidth">
          <select v-model="releaseMonth">
            <option></option>
            <option v-for="t in filters.release_month" :key="t">{{ t }}</option>
          </select>
        </div>
      </div>
      <div class="control">
        <button type="submit" class="button is-light" @click="clearReleaseMonth">
          <b-icon pack="fas" icon="times" size="is-small"></b-icon>
        </button>
      </div>
    </b-field>

    <b-field label="Folder" label-position="on-border" :addons="true" class="field-extra">
      <div class="control is-expanded">
        <div class="select is-fullwidth">
          <select v-model="volume">
            <option :value="0"></option>
            <option v-for="t in filters.volumes" :key="t.id" :value="t.id">{{ t.path }}</option>
          </select>
        </div>
      </div>
      <div class="control">
        <button type="submit" class="button is-light" @click="clearVolume">
          <b-icon pack="fas" icon="times" size="is-small"></b-icon>
        </button>
      </div>
    </b-field>

    <div class="is-divider" data-content="Filters"></div>

    <div v-if="Object.keys(filters).length !== 0">
      <b-field label="Cast" label-position="on-border" class="field-extra">
        <b-taginput v-model="cast" autocomplete :data="filteredCast" @typing="getFilteredCast">
          <template slot-scope="props">{{ props.option }}</template>
          <template slot="empty">No matching cast</template>
        </b-taginput>
      </b-field>

      <b-field label="Site" label-position="on-border" class="field-extra">
        <b-taginput v-model="sites" autocomplete :data="filteredSites" @typing="getFilteredSites">
          <template slot-scope="props">{{ props.option }}</template>
          <template slot="empty">No matching sites</template>
        </b-taginput>
      </b-field>

      <b-field label="Tags" label-position="on-border" class="field-extra">
        <b-taginput v-model="tags" autocomplete :data="filteredTags" @typing="getFilteredTags">
          <template slot-scope="props">{{ props.option }}</template>
          <template slot="empty">No matching tags</template>
        </b-taginput>
      </b-field>

      <b-field label="Cuepoint" label-position="on-border" class="field-extra">
        <b-taginput v-model="cuepoint" allow-new>
          <template slot-scope="props">{{ props.option }}</template>
          <template slot="empty">No matching cuepoints</template>
        </b-taginput>
      </b-field>

    </div>
    <div class="is-divider" data-content="Actor Also Known As groups"></div>
    <b-field>
      <b-tooltip position="is-right" label="New Aka Group. Select 2 or more actors in the Cast filter" multilined :delay="200">
        <button class="button is-small is-outlined" @click="createAkaGroup" :disabled="disableNewAkaGroup">
          <b-icon pack="mdi" icon="account-multiple-plus-outline"></b-icon>
        </button>
      </b-tooltip>
      <b-tooltip position="is-right" label="Select the Aka Group to delete in the Cast Filter" multilined :delay="200">
        <button class="button is-small is-outlined" @click="deleteAkaGroup" :disabled="disableDeleteAkaGroup">
          <b-icon pack="mdi" icon="delete-outline"></b-icon>
        </button>
      </b-tooltip>
      <b-tooltip position="is-bottom" label="Add Cast to Aka Group. Select the Aka group and Actors to add in the Cast Filter" multilined :delay="200">
        <button class="button is-small is-outlined" @click="addToAkaGroup" :disabled="disableAddToAkaGroup">
          <b-icon pack="mdi" icon="account-plus-outline"></b-icon>
        </button>
      </b-tooltip>
      <b-tooltip position="is-bottom" label="Remove Cast from Aka Group. Select the Aka group and Actors to remove in the Cast Filter" multilined :delay="200">
        <button class="button is-small is-outlined" @click="removeFromAkaGroup" :disabled="disableRemoveFromAkaGroup">
          <b-icon pack="mdi" icon="account-minus-outline"></b-icon>
        </button>
      </b-tooltip>

    </b-field>
  </div>
</template>

<script>
import SavedSearch from './SavedSearch'
import ky from 'ky'

export default {
  name: 'Filters',
  components: { SavedSearch },
  mounted () {
    this.$store.dispatch('sceneList/filters')
  },
  data () {
    return {
      filteredCast: [],
      filteredSites: [],
      filteredTags: []
    }
  },
  methods: {
    reloadList () {
      this.$router.push({
        name: 'scenes',
        query: {
          q: this.$store.getters['sceneList/filterQueryParams']
        }
      })
    },
    getFilteredCast (text) {
      this.filteredCast = this.filters.cast.filter(option => (
        option.toString().toLowerCase().indexOf(text.toLowerCase()) >= 0 &&
        !this.cast.some(entry => entry.toString() === option.toString())
      ))      
    },
    getFilteredSites (text) {
      this.filteredSites = this.filters.sites.filter(option => (
        option.toString().toLowerCase().indexOf(text.toLowerCase()) >= 0 &&
        !this.sites.some(entry => entry.toString() === option.toString())
      ))
    },
    getFilteredTags (text) {
      this.filteredTags = this.filters.tags.filter(option => (
        option.toString().toLowerCase().indexOf(text.toLowerCase()) >= 0 &&
        !this.tags.some(entry => entry.toString() === option.toString())
      ))
    },
    clearReleaseMonth () {
      this.$store.state.sceneList.filters.releaseMonth = ''
      this.reloadList()
    },
    clearVolume () {
      this.$store.state.sceneList.filters.volume = 0
      this.$store.dispatch('sceneList/filters')
      this.reloadList()
    },
    createAkaGroup () {
      this.$store.state.sceneList.isLoading = true
      ky.post('/api/aka/create', {json: {actorList: this.cast}}).json().then(data => {
        this.cast.push(data.akas.aka_actor.name)
        this.$store.dispatch('sceneList/filters')
        this.reloadList()
        if (data.status != '') {
          this.$buefy.toast.open({message: `Warning:  ${data.status}`, type: 'is-warning', duration: 5000})
        }
        this.$store.state.sceneList.isLoading = false
      })
    },
    deleteAkaGroup () {
      this.$store.state.sceneList.isLoading = true
      ky.post('/api/aka/delete', {json: {name: this.cast[0]}}).json().then(data => {
        this.cast = []
        this.$store.dispatch('sceneList/filters')
        this.reloadList()
        this.$store.state.sceneList.isLoading = false
      })       
    },
    addToAkaGroup () {
      this.$store.state.sceneList.isLoading = true
      ky.post('/api/aka/add', {json: {actorList: this.cast}}).json().then(data => {        
        // delete old aka & add new name
        this.cast = this.cast.filter(e => !e.startsWith("aka:")) 
        this.cast.push(data.akas.aka_actor.name) 
        this.$store.dispatch('sceneList/filters')       
        this.reloadList()        
        if (data.status != '') {
          this.$buefy.toast.open({message: `Warning:  ${data.status}`, type: 'is-warning', duration: 5000})
        }
        this.$store.state.sceneList.isLoading = false
      })
      
    },
    removeFromAkaGroup () {
      this.$store.state.sceneList.isLoading = true
      ky.post('/api/aka/remove', {json: {actorList: this.cast}}).json().then(data => {        
        // delete old aka & add new name
        this.cast = this.cast.filter(e => !e.startsWith("aka:")) 
        this.cast.push(data.akas.aka_actor.name)
        this.$store.dispatch('sceneList/filters')
        this.reloadList()
        if (data.status != '') {
          this.$buefy.toast.open({message: `Warning:  ${data.status}`, type: 'is-warning', duration: 5000})
        }
        this.$store.state.sceneList.isLoading = false
      })
    },
  },
  computed: {
    filters () {
      return this.$store.state.sceneList.filterOpts
    },
    lists: {
      get () {
        return this.$store.state.sceneList.filters.lists
      },
      set (value) {
        this.$store.state.sceneList.filters.lists = value
        this.reloadList()
      }
    },
    releaseMonth: {
      get () {
        return this.$store.state.sceneList.filters.releaseMonth
      },
      set (value) {
        this.$store.state.sceneList.filters.releaseMonth = value
        this.reloadList()
      }
    },
    volume: {
      get () {
        return this.$store.state.sceneList.filters.volume
      },
      set (value) {
        this.$store.state.sceneList.filters.volume = value
        this.reloadList()
      }
    },
    cast: {
      get () {
        return this.$store.state.sceneList.filters.cast
      },
      set (value) {
        this.$store.state.sceneList.filters.cast = value
        this.reloadList()
      }
    },
    sites: {
      get () {
        return this.$store.state.sceneList.filters.sites
      },
      set (value) {
        this.$store.state.sceneList.filters.sites = value
        this.reloadList()
      }
    },
    tags: {
      get () {
        return this.$store.state.sceneList.filters.tags
      },
      set (value) {
        this.$store.state.sceneList.filters.tags = value
        this.reloadList()
      }
    },
    cuepoint: {
      get () {
        return this.$store.state.sceneList.filters.cuepoint
      },
      set (value) {
        this.$store.state.sceneList.filters.cuepoint = value
        this.reloadList()
      }
    },
    sort: {
      get () {
        return this.$store.state.sceneList.filters.sort
      },
      set (value) {
        this.$store.state.sceneList.filters.sort = value
        this.reloadList()
      }
    },
    isWatched: {
      get () {
        return this.$store.state.sceneList.filters.isWatched
      },
      set (value) {
        this.$store.state.sceneList.filters.isWatched = value
        this.reloadList()
      }
    },
    disableNewAkaGroup() {
      let akaCastCnt = 0
      let actorCnt = 0       
 
      for (var i = 0; i < this.cast.length; i++) {        
        if (this.cast[i].startsWith("aka:")) {
          akaCastCnt++
        } else {
          actorCnt++
        }
      }
      // you can create a new group from a list of actors (more than one)
      return akaCastCnt == 0 && actorCnt > 1 ? false : true
    },
    disableDeleteAkaGroup() {
      let akaCastCnt = 0
      let actorCnt = 0       
 
      for (var i = 0; i < this.cast.length; i++) {        
        if (this.cast[i].startsWith("aka:")) {
          akaCastCnt++
        } else {
          actorCnt++
        }
      }

      // you can only delete a group when it is the only thing selected      
      return akaCastCnt == 1 && actorCnt == 0 > 1 ? false : true
    },
    disableAddToAkaGroup() {
      let akaCastCnt = 0
      let actorCnt = 0       
 
      for (var i = 0; i < this.cast.length; i++) {        
        if (this.cast[i].startsWith("aka:")) {
          akaCastCnt++
        } else {
          actorCnt++
        }
      }

      // you can add to a group if you select one group and one or more actors
      return akaCastCnt == 1 && actorCnt > 0 ? false : true
    },
    disableRemoveFromAkaGroup() {
      let akaCastCnt = 0
      let actorCnt = 0       
 
      for (var i = 0; i < this.cast.length; i++) {        
        if (this.cast[i].startsWith("aka:")) {
          akaCastCnt++
        } else {
          actorCnt++
        }
      }

      // you can remove from a group if you select one group and one or more actors
      return akaCastCnt == 1 && actorCnt > 0 ? false : true

    }
  }
}
</script>

<style lang="scss" scoped>
@import "~bulma-extensions/bulma-divider/dist/css/bulma-divider.min.css";

.is-gapless div.control {
  margin: 0.1rem;
}

.is-divider {
  margin: 1.5rem 0;
}

.field-extra {
  margin-bottom: 1.1em !important;
}
</style>
