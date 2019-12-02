<template>
  <div>
    <div class="field">
      <label class="label">{{$t("State")}}</label>
      <div class="control is-expanded">
        <div class="select is-fullwidth">
          <select v-model="dlState">
            <option value="any">{{$t("Any")}}</option>
            <option value="available">{{$t("Available right now")}}</option>
            <option value="downloaded">{{$t("Downloaded")}}</option>
            <option value="missing">{{$t("Not downloaded")}}</option>
          </select>
        </div>
      </div>
    </div>

    <div class="field">
      <label class="label">List</label>
      <b-field>
        <b-checkbox-button v-model="lists" native-value="watchlist" type="is-primary">
          <b-icon pack="mdi" icon="calendar-check" size="is-small"/>
          <span>{{$t("Watchlist")}}</span>
        </b-checkbox-button>
        <b-checkbox-button v-model="lists" native-value="favourite" type="is-danger">
          <b-icon pack="mdi" icon="heart" size="is-small"/>
          <span>{{$t("Favourite")}}</span>
        </b-checkbox-button>
      </b-field>
    </div>

    <label class="label">{{$t("Sort by")}}</label>
    <div class="field has-addons">
      <div class="control is-expanded">
        <div class="select is-fullwidth">
          <select v-model="sort">
            <option value="release_desc">↓ {{$t("Release date")}}</option>
            <option value="release_asc">↑ {{$t("Release date")}}</option>
            <option value="added_desc">↓ {{$t("File added date")}}</option>
            <option value="added_asc">↑ {{$t("File added date")}}</option>
            <option value="rating_desc">↓ {{$t("Rating")}}</option>
            <option value="rating_asc">↑ {{$t("Rating")}}</option>
            <option value="scene_added_desc">↓ {{$t("Scene added date")}}</option>
            <option value="scene_updated_desc">↓ {{$t("Scene updated date")}}</option>
            <option value="last_opened">↻ {{$t("Recently viewed")}}</option>
            <option value="random">↯ {{$t("Random")}}</option>
          </select>
        </div>
      </div>
    </div>

    <label class="label">Watch status</label>
    <div class="field has-addons">
      <div class="control is-expanded">
        <div class="select is-fullwidth">
          <select v-model="isWatched">
            <option :value="null">Everything</option>
            <option :value="true">Watched</option>
            <option :value="false">Unwatched</option>
          </select>
        </div>
      </div>
    </div>

    <div v-if="Object.keys(filters).length !== 0">
      <label class="label">Release date</label>
      <div class="field has-addons">
        <div class="control is-expanded">
          <div class="select is-fullwidth">
            <select v-model="releaseMonth">
              <option></option>
              <option v-for="t in filters.release_month" :key="t">{{t}}</option>
            </select>
          </div>
        </div>
        <div class="control">
          <button type="submit" class="button is-light" @click="clearReleaseMonth">
            <b-icon pack="fas" icon="times" size="is-small"></b-icon>
          </button>
        </div>
      </div>


      <label class="label">Cast</label>
      <div class="field">
        <b-taginput v-model="cast" autocomplete :data="filteredCast" @typing="getFilteredCast">
          <template slot-scope="props">{{props.option}}</template>
          <template slot="empty">No matching cast</template>
        </b-taginput>
      </div>

      <label class="label">Site</label>
      <div class="field">
        <b-taginput v-model="sites" autocomplete :data="filteredSites" @typing="getFilteredSites">
          <template slot-scope="props">{{props.option}}</template>
          <template slot="empty">No matching sites</template>
        </b-taginput>
      </div>

      <label class="label">Tags</label>
      <div class="field">
        <b-taginput v-model="tags" autocomplete :data="filteredTags" @typing="getFilteredTags">
          <template slot-scope="props">{{props.option}}</template>
          <template slot="empty">No matching tags</template>
        </b-taginput>
      </div>

      <label class="label">Cuepoint</label>
      <div class="field">
        <b-taginput v-model="cuepoint" allow-new>
          <template slot-scope="props">{{props.option}}</template>
          <template slot="empty">No matching cuepoints</template>
        </b-taginput>
      </div>

    </div>
  </div>
</template>

<script>
  export default {
    name: "Filters",
    mounted() {
      this.$store.dispatch("sceneList/filters");
    },
    data() {
      return {
        filteredCast: [],
        filteredSites: [],
        filteredTags: [],
      }
    },
    methods: {
      reload() {
        this.$router.push({
          name: 'scenes',
          query: {
            q: this.$store.getters['sceneList/filterQueryParams']
          }
        });
      },
      getFilteredCast(text) {
        this.filteredCast = this.filters.cast.filter((option) => {
          return option.toString().toLowerCase().indexOf(text.toLowerCase()) >= 0
        })
      },
      getFilteredSites(text) {
        this.filteredSites = this.filters.sites.filter((option) => {
          return option.toString().toLowerCase().indexOf(text.toLowerCase()) >= 0
        })
      },
      getFilteredTags(text) {
        this.filteredTags = this.filters.tags.filter((option) => {
          return option.toString().toLowerCase().indexOf(text.toLowerCase()) >= 0
        })
      },
      clearReleaseMonth() {
        this.$store.state.sceneList.filters.releaseMonth = "";
        this.reload();
      },
    },
    computed: {
      filters() {
        return this.$store.state.sceneList.filterOpts;
      },
      lists: {
        get() {
          return this.$store.state.sceneList.filters.lists;
        },
        set(value) {
          this.$store.state.sceneList.filters.lists = value;
          this.reload();
        }
      },
      dlState: {
        get() {
          return this.$store.state.sceneList.filters.dlState;
        },
        set(value) {
          this.$store.state.sceneList.filters.dlState = value;

          switch (this.$store.state.sceneList.filters.dlState) {
            case "any":
              this.$store.state.sceneList.filters.isAvailable = null;
              this.$store.state.sceneList.filters.isAccessible = null;
              break;
            case "available":
              this.$store.state.sceneList.filters.isAvailable = true;
              this.$store.state.sceneList.filters.isAccessible = true;
              break;
            case "downloaded":
              this.$store.state.sceneList.filters.isAvailable = true;
              this.$store.state.sceneList.filters.isAccessible = null;
              break;
            case "missing":
              this.$store.state.sceneList.filters.isAvailable = false;
              this.$store.state.sceneList.filters.isAccessible = null;
              break;
          }

          this.reload();
        }
      },
      releaseMonth: {
        get() {
          return this.$store.state.sceneList.filters.releaseMonth;
        },
        set(value) {
          this.$store.state.sceneList.filters.releaseMonth = value;
          this.reload();
        }
      },
      cast: {
        get() {
          return this.$store.state.sceneList.filters.cast;
        },
        set(value) {
          this.$store.state.sceneList.filters.cast = value;
          this.reload();
        }
      },
      sites: {
        get() {
          return this.$store.state.sceneList.filters.sites;
        },
        set(value) {
          this.$store.state.sceneList.filters.sites = value;
          this.reload();
        }
      },
      tags: {
        get() {
          return this.$store.state.sceneList.filters.tags;
        },
        set(value) {
          this.$store.state.sceneList.filters.tags = value;
          this.reload();
        }
      },
      cuepoint: {
        get() {
          return this.$store.state.sceneList.filters.cuepoint;
        },
        set(value) {
          this.$store.state.sceneList.filters.cuepoint = value;
          this.reload();
        }
      },
      sort: {
        get() {
          return this.$store.state.sceneList.filters.sort;
        },
        set(value) {
          this.$store.state.sceneList.filters.sort = value;
          this.reload();
        }
      },
      isWatched: {
        get() {
          return this.$store.state.sceneList.filters.isWatched;
        },
        set(value) {
          this.$store.state.sceneList.filters.isWatched = value;
          this.reload();
        }
      },
    }
  }
</script>
