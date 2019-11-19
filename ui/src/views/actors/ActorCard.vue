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
  .is-one-fifth figure {
    height: 177px;
    overflow: hidden;
  }
  .is-one-quarter figure {
    height: 227px;
    overflow: hidden;
  }
  .is-one-third figure {
    height: 311px;
    overflow: hidden;
  }

  .transparent {
    opacity: 0.35;
  }

  .button {
    margin-right: 3px;
  }
</style>
