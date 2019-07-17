<template>
  <div>
    <div class="field">
      <label class="label">Cover size</label>
      <input type=range v-model="cardSize" min=1 max=3>
    </div>

    <div class="field">
      <label class="label">State</label>
      <div class="control is-expanded">
        <div class="select is-fullwidth">
          <select v-model="dlState">
            <option value="any">Any</option>
            <option value="available">Available right now</option>
            <option value="downloaded">Downloaded</option>
            <option value="missing">Not downloaded</option>
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
      <div class="field has-addons">
        <div class="control is-expanded">
          <div class="select is-fullwidth">
            <select v-model="cast">
              <option></option>
              <option v-for="t in filters.cast" :key="t">{{t}}</option>
            </select>
          </div>
        </div>
        <div class="control">
          <button type="submit" class="button is-light" @click="clearCast">
            <b-icon pack="fas" icon="times" size="is-small"></b-icon>
          </button>
        </div>
      </div>

      <label class="label">Site</label>
      <div class="field has-addons">
        <div class="control is-expanded">
          <div class="select is-fullwidth">
            <select v-model="site">
              <option></option>
              <option v-for="t in filters.sites" :key="t">{{t}}</option>
            </select>
          </div>
        </div>
        <div class="control">
          <button type="submit" class="button is-light" @click="clearSite">
            <b-icon pack="fas" icon="times" size="is-small"></b-icon>
          </button>
        </div>
      </div>

      <label class="label">Tags</label>
      <div class="field has-addons">
        <div class="control is-expanded">
          <div class="select is-fullwidth">
            <select v-model="tag">
              <option></option>
              <option v-for="t in filters.tags" :key="t">{{t}}</option>
            </select>
          </div>
        </div>
        <div class="control">
          <button type="submit" class="button is-light" @click="clearTag">
            <b-icon pack="fas" icon="times" size="is-small"></b-icon>
          </button>
        </div>
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
    methods: {
      clearReleaseMonth() {
        this.$store.state.sceneList.filters.releaseMonth = "";
        this.$store.dispatch("sceneList/load", {offset: 0});
      },
      clearCast() {
        this.$store.state.sceneList.filters.cast = "";
        this.$store.dispatch("sceneList/load", {offset: 0});
      },
      clearSite() {
        this.$store.state.sceneList.filters.site = "";
        this.$store.dispatch("sceneList/load", {offset: 0});
      },
      clearTag() {
        this.$store.state.sceneList.filters.tag = "";
        this.$store.dispatch("sceneList/load", {offset: 0});
      },
    },
    computed: {
      filters() {
        return this.$store.state.sceneList.filterOpts;
      },
      cardSize: {
        get() {
          return this.$store.state.sceneList.filters.cardSize;
        },
        set(value) {
          this.$store.state.sceneList.filters.cardSize = value;
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
              this.$store.state.sceneList.filters.isAvailable = "";
              this.$store.state.sceneList.filters.isAccessible = "";
              break;
            case "available":
              this.$store.state.sceneList.filters.isAvailable = "1";
              this.$store.state.sceneList.filters.isAccessible = "1";
              break;
            case "downloaded":
              this.$store.state.sceneList.filters.isAvailable = "1";
              this.$store.state.sceneList.filters.isAccessible = "";
              break;
            case "missing":
              this.$store.state.sceneList.filters.isAvailable = "0";
              this.$store.state.sceneList.filters.isAccessible = "";
              break;
          }

          this.$store.dispatch("sceneList/load", {offset: 0});
          this.$store.dispatch("sceneList/filters");
        }
      },
      releaseMonth: {
        get() {
          return this.$store.state.sceneList.filters.releaseMonth;
        },
        set(value) {
          this.$store.state.sceneList.filters.releaseMonth = value;
          this.$store.dispatch("sceneList/load", {offset: 0});
        }
      },
      cast: {
        get() {
          return this.$store.state.sceneList.filters.cast;
        },
        set(value) {
          this.$store.state.sceneList.filters.cast = value;
          this.$store.dispatch("sceneList/load", {offset: 0});
        }
      },
      site: {
        get() {
          return this.$store.state.sceneList.filters.site;
        },
        set(value) {
          this.$store.state.sceneList.filters.site = value;
          this.$store.dispatch("sceneList/load", {offset: 0});
        }
      },
      tag: {
        get() {
          return this.$store.state.sceneList.filters.tag;
        },
        set(value) {
          this.$store.state.sceneList.filters.tag = value;
          this.$store.dispatch("sceneList/load", {offset: 0});
        }
      },
    }
  }
</script>
