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

      <watchlist-button :item="item"/>
      <favourite-button :item="item"/>

      <a class="button is-outlined is-small" v-if="item.is_watched">
        <b-icon pack="far" icon="eye" size="is-small"/>
      </a>

      <span class="is-pulled-right" style="font-size:11px;text-align:right;">
        <a :href="item.scene_url" target="_blank">{{item.site}}</a><br/>
        <span v-if="item.release_date !== '0001-01-01T00:00:00Z'">
          {{format(parseISO(item.release_date), "yyyy-MM-dd")}}
        </span>
      </span>
    </div>
  </div>
</template>

<script>
  import {format, parseISO} from "date-fns";
  import VueLoadImage from "vue-load-image";
  import WatchlistButton from "../../components/WatchlistButton";
  import FavouriteButton from "../../components/FavouriteButton";

  export default {
    name: "SceneCard",
    props: {item: Object},
    components: {VueLoadImage, WatchlistButton, FavouriteButton},
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
