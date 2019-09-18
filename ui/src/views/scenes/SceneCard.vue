<template>
  <div class="card is-shadowless">
    <div class="card-image">
      <figure class="image" @click="showDetails(item)">
        <vue-load-image>
          <img slot="image" :src="getImageURL(item.cover_url)" v-bind:class="{'transparent': !item.is_available}"/>
          <img slot="preloader" src="/ui/images/blank.png"/>
          <img slot="error" src="/ui/images/blank.png"/>
        </vue-load-image>
      </figure>
    </div>

    <div style="padding-top:4px;">
      <a class="button is-danger is-small"
         @click="toggleList(item.scene_id, 'favourite')"
         v-show="item.favourite">
        <b-icon pack="fas" icon="heart" size="is-small"></b-icon>
      </a>
      <a class="button is-danger is-outlined is-small"
         @click="toggleList(item.scene_id, 'favourite')"
         v-show="!item.favourite">
        <b-icon pack="far" icon="heart" size="is-small"></b-icon>
      </a>

      <a class="button is-primary is-small"
         @click="toggleList(item.scene_id, 'watchlist')"
         v-show="item.watchlist">
        <b-icon pack="fas" icon="calendar-check" size="is-small"></b-icon>
      </a>
      <a class="button is-primary is-outlined is-small"
         @click="toggleList(item.scene_id, 'watchlist')"
         v-show="!item.watchlist">
        <b-icon pack="far" icon="calendar-check" size="is-small"></b-icon>
      </a>

      <a class="button is-outlined is-small" v-if="item.is_watched">
        <b-icon pack="far" icon="eye" size="is-small"/>
      </a>

      <span class="is-pulled-right" style="font-size:11px;text-align:right;">
        <a :href="item.scene_url" target="_blank">{{item.site}}</a><br/>
        {{format(parseISO(item.release_date), "yyyy-MM-dd")}}
      </span>
    </div>
  </div>
</template>

<script>
  import {format, parseISO} from "date-fns";
  import VueLoadImage from "vue-load-image";

  export default {
    name: "SceneCard",
    props: {item: Object},
    components: {VueLoadImage,},
    data() {
      return {format, parseISO}
    },
    methods: {
      getImageURL(u) {
        if (u.startsWith("http")) {
          return "/img/700x/" + u.replace("://", ":/");
        } else {
          return u;
        }
      },
      toggleList(scene_id, list) {
        this.$store.commit("sceneList/toggleSceneList", {scene_id: scene_id, list: list});
      },
      showDetails(scene) {
        this.$store.commit("overlay/showDetails", {scene: scene});
      }
    }
  }
</script>

<style scoped>
  .transparent {
    opacity: 0.35;
  }

  .button {
    margin-right: 3px;
  }
</style>