<template>
  <div>
    <div class="is-divider" data-content="Saved lists" style="margin-top:0.8em;"></div>

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
  </div>
</template>

<script>
import SavedSearch from './SavedSearch'

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
      this.filteredCast = this.filters.cast.filter((option) => {
        return option.toString().toLowerCase().indexOf(text.toLowerCase()) >= 0
      })
    },
    getFilteredSites (text) {
      this.filteredSites = this.filters.sites.filter((option) => {
        return option.toString().toLowerCase().indexOf(text.toLowerCase()) >= 0
      })
    },
    getFilteredTags (text) {
      this.filteredTags = this.filters.tags.filter((option) => {
        return option.toString().toLowerCase().indexOf(text.toLowerCase()) >= 0
      })
    },
    clearReleaseMonth () {
      this.$store.state.sceneList.filters.releaseMonth = ''
      this.reloadList()
    },
    clearVolume () {
      this.$store.state.sceneList.filters.volume = 0
      this.reloadList()
    }
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
