<template>
  <div class="card is-shadowless">
    <div class="card-image">
      <figure class="image" @click="showScenes(item.name)">
        <vue-load-image>
          <img slot="image" :src="getImageURL(item.image_url)"/>
          <img slot="preloader" src="/ui/images/blank.png"/>
          <img slot="error" src="/ui/images/blank.png"/>
        </vue-load-image>
      </figure>
    </div>

    <div>
      <span v-if="item.twitter" class="is-pulled-left">
        <a :href=item.twitter target="_blank">
          <b-icon pack="fab" icon="twitter-square" type="is-primary"/>
        </a>
      </span>
      <span v-if="item.instagram" class="is-pulled-left">
        <a :href=item.instagram target="_blank">
          <b-icon pack="fab" icon="instagram" type="is-primary"/>
        </a>
      </span>
      <span v-if="item.reddit" class="is-pulled-left">
        <a :href=item.reddit target="_blank">
          <b-icon pack="fab" icon="reddit-square" type="is-primary"/>
        </a>
      </span>
      <span v-if="item.facebook" class="is-pulled-left">
        <a :href=item.facebook target="_blank">
          <b-icon pack="fab" icon="facebook-square" type="is-primary"/>
        </a>
      </span>
    </div>
    <div style="padding-top:4px;">
      <span class="is-pulled-right" style="font-size:11px;text-align:right;">
        <a :href="item.homepage_url" target="_blank">{{item.name}}</a>
      </span>
    </div>
  </div>
</template>

<script>
  import VueLoadImage from "vue-load-image";

  export default {
    name: "ActorCard",
    props: {item: Object},
    components: {VueLoadImage},
    methods: {
      getImageURL(u) {
        if (u.startsWith("http")) {
          return "/img/700x/" + u.replace("://", ":/");
        } else {
          return u;
        }
      },
      showScenes(actor) {
        this.$store.state.sceneList.filters.cast = [actor];
        this.$store.state.sceneList.filters.sites = [];
        this.$store.state.sceneList.filters.tags = [];
        this.$router.push({
          name: 'scenes',
          query: {q: this.$store.getters['sceneList/filterQueryParams']}
        });
      },
    }
  }
</script>

<style scoped>
  figure {
    height: 0px;
    padding-top: 100%;
    overflow: hidden;
  }

  figure img {
    position: absolute;
    top: 0;
    left: 0;
  }

  .transparent {
    opacity: 0.35;
  }

  .button {
    margin-right: 3px;
  }
</style>
