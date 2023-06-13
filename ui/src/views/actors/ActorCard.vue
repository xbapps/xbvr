<template>
  <div class="card is-shadowless">
    <div class="card-image">
      <div class="bbox"
           v-bind:style="{backgroundImage: `url(${getImageURL(actor.image_url)})`, backgroundSize: 'contain', backgroundPosition: 'center', backgroundRepeat: 'no-repeat', opacity:isAvailable(actor) ? 1.0 : 0.4}"
           @click="showDetails(actor)"
           @mouseover="preview = true"
           @mouseleave="preview = false">
      </div>
        <div class="overlay align-bottom-left">
         </div>
    </div>

    <div style="padding-top:4px;">
      <div class="scene_title">{{actor.name}}</div>
      <a v-if="colleague!=undefined" class="button is-info is-outlined is-small"
        @click="showColleague(actor.name,colleague)"
        :title="'Show Scenes with ' + actor.name">
        <b-icon pack="mdi" :icon="'movie-outline'" size="is-small"/>
      </a>
      <actor-favourite-button :actor="actor" v-if="this.$store.state.optionsWeb.web.sceneFavourite"/>
      <actor-watchlist-button :actor="actor" v-if="this.$store.state.optionsWeb.web.sceneWatchlist"/>
      <actor-edit-button :actor="actor"/>&nbsp;
      <b-tooltip :label="$t('Your rating')" :delay="500">
      <b-tag type="is-warning" v-if="actor.star_rating != 0 " size="is-small" style="height:30px;">
        <b-icon pack="mdi" icon="star" size="is-small"/>
        {{actor.star_rating}}
      </b-tag>
      </b-tooltip>
      <b-tooltip :label="$t('Average rating of scenes')" :delay="500">
      <b-tag type="is-primary" v-if="actor.scene_rating_average != 0 " style="height:30px;">
        <b-icon pack="mdi" icon="star" size="is-small"/>
        {{Math.round(actor.scene_rating_average * 4) / 4}}
      </b-tag>
      </b-tooltip>

      <span class="is-pulled-right" style="font-size:11px;text-align:right;">                
        <b-field grouped>
          <span v-if="actor.birth_date != '0001-01-01T00:00:00Z'">{{format(parseISO(actor.birth_date), "yyyy-MM-dd")}}</span>
        <vue-load-image>
          <img slot="image" :src="getImageURL('https://flagcdn.com/' + actor.nationality.toLowerCase() +'.svg')" style="height:10px;border: 1px solid black;margin-left:0.5em;" />
        </vue-load-image>
        </b-field>
      </span>
    </div>
  </div>
</template>

<script>
import { format, parseISO } from 'date-fns'
import ActorFavouriteButton from '../../components/ActorFavouriteButton'
import ActorWatchlistButton from '../../components/ActorWatchlistButton'
import ActorEditButton from '../../components/ActorEditButton'
import VueLoadImage from 'vue-load-image'
import { tr } from 'date-fns/locale'

export default {
  name: 'ActorCard',
  props: { actor: Object, colleague: String },
   components: {ActorFavouriteButton, ActorWatchlistButton, VueLoadImage, ActorEditButton},
  data () {
    return {
      preview: false,
      format,
      parseISO
    }
  },
  computed: {
  },
  methods: {
    getImageURL (u) {
      if (u=='' || u == undefined) {
        return "/ui/images/blank_female_profile.png"
      }
      if (u.startsWith('http')) {
        return '/img/700x/' + u.replace('://', ':/')
      } else {
        return u
      }
    },
    showDetails (actor) {
      this.$store.commit('overlay/showActorDetails', { actor: actor })
    },
    isAvailable(actor) {
      let index = actor.scenes.findIndex(scene => scene.is_available == 1);
      if (index == -1) {
        return false
      }
      return true
    },
    showColleague (main_actor, colleague) {      
      this.$store.state.sceneList.filters.cast = ["&" + main_actor , "&"+ colleague]
      this.$store.state.sceneList.filters.sites = []
      this.$store.state.sceneList.filters.tags = []
      this.$store.state.sceneList.filters.attributes = []
      this.$store.state.actorList.filters.dlState = "Any"
      this.$router.push({
        name: 'scenes',
        query: { q: this.$store.getters['sceneList/filterQueryParams'] }
      })
      this.$store.commit('overlay/hideActorDetails')
    },

  },
}
</script>

<style scoped>
  .button {
    margin-right: 3px;
  }

  .bbox {
    flex: 1 0 calc(25%);
    background: #f0f0f0;
    display: flex;
    align-items: center;
    justify-content: center;
    overflow: hidden;
    padding: 0;
    line-height: 0;
  }

  .bbox:not(:hover) > video {
    display: none;
  }

  video {
    object-fit: cover;
    position: absolute;
    width: 100%;
    height: 100%;
  }

  .overlay {
   position: absolute;
  bottom: 0;
  right: 0;
  display: flex;  
  padding: 5px;
  max-width: 5px;
  }

  .align-bottom-left {
    align-items: flex-end;
    justify-content: flex-end;
  }

  .bbox:after {
    content: '';
    display: block;
    padding-bottom: 100%;
  }

  .tag {
    margin-left: 0.1em;
  }

  .scene_title {
    font-size: 12px;
    text-align: right;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
</style>
