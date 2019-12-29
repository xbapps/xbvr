<template>
  <div class="column">
    <b-loading :is-full-page="true" :active.sync="isLoading"></b-loading>

    <div class="columns is-multiline is-full">
      <div class="column">
        <strong>{{total}} results</strong>
      </div>
      <div class="column">
        <div class="is-pulled-right">
          <b-field>
            <span class="list-header-label">{{$t('Card size')}}</span>
            <b-radio-button v-model="cardSize" native-value="1" size="is-small">
              S
            </b-radio-button>
            <b-radio-button v-model="cardSize" native-value="2" size="is-small">
              M
            </b-radio-button>
            <b-radio-button v-model="cardSize" native-value="3" size="is-small">
              L
            </b-radio-button>
          </b-field>
        </div>
      </div>
    </div>

    <div class="columns is-full">
      <div class="column">
        <b-select size="is-small" @input="setPlaylist">
          <option v-for="(obj, idx) in playlists" :value="obj.id" :key="obj.id">
            {{ obj.name }}
          </option>
        </b-select>

        <b-button size="is-small" @click="isPlaylistModalActive=true">Save as smart playlist</b-button>
      </div>
    </div>

    <div class="columns is-multiline">
      <div :class="['column', 'is-multiline', cardSizeClass]"
           v-for="item in items" :key="item.id">
        <SceneCard :item="item"/>
      </div>
    </div>

    <div class="column is-full" v-if="items.length < total">
      <a class="button is-fullwidth" v-on:click="loadMore()">{{$t('Load more')}}</a>
    </div>

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
  import SceneCard from "./SceneCard";
  import ky from "ky";

  export default {
    name: "List",
    data() {
      return {
        playlistName: "",
        playlistDeoEnabled: false,
        isPlaylistModalActive: false,
      }
    },
    components: {SceneCard},
    computed: {
      cardSize: {
        get() {
          return this.$store.state.sceneList.filters.cardSize;
        },
        set(value) {
          this.$store.state.sceneList.filters.cardSize = value;
        }
      },
      cardSizeClass() {
        switch (this.$store.state.sceneList.filters.cardSize) {
          case "1":
            return "is-one-fifth";
          case "2":
            return "is-one-quarter";
          case "3":
            return "is-one-third";
          default:
            return "is-one-fifth";
        }
      },
      playlists() {
        return this.$store.state.sceneList.playlists;
      },
      isLoading() {
        return this.$store.state.sceneList.isLoading;
      },
      items() {
        return this.$store.state.sceneList.items;
      },
      total() {
        return this.$store.state.sceneList.total;
      }
    },
    methods: {
      setPlaylist(val) {
        const obj = this.playlists.find(item => item.id === val);
        this.$router.push({
          name: 'scenes',
          query: {
            q: this.$store.getters['sceneList/getQueryParamsFromObject'](obj.search_params)
          }
        });

        this.$store.dispatch("sceneList/load", {offset: 0});
      },
      async savePlaylist() {
        await ky.post(`/api/playlist`, {
          json: {
            name: this.playlistName,
            is_deo_enabled: this.playlistDeoEnabled,
            is_smart: true,
            search_params: JSON.stringify(this.$store.state.sceneList.filters),
          }
        });
        this.isPlaylistModalActive = false;
        this.$store.dispatch("sceneList/filters");
      },
      async loadMore() {
        this.$store.dispatch("sceneList/load", {offset: this.$store.state.sceneList.offset});
      }
    }
  }
</script>

<style scoped>
  .list-header-label {
    padding-right: 1em;
  }
</style>
