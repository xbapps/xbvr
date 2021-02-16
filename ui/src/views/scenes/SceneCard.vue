<template>
  <div class="card is-shadowless">
    <div class="card-image">
      <div class="bbox"
           v-bind:style="{backgroundImage: `url(${getImageURL(item.cover_url)})`, backgroundSize: 'contain', backgroundPosition: 'center', backgroundRepeat: 'no-repeat', opacity:item.is_available ? 1.0 : 0.4}"
           @click="showDetails(item)"
           @mouseover="preview = true"
           @mouseleave="preview = false">
        <video v-if="preview && item.has_preview" :src="`/api/dms/preview/${item.scene_id}`" autoplay loop></video>
        <div v-else>
          <div class="overlay align-bottom-left">
            <div style="padding: 5px">
              <b-tag v-if="item.is_watched">
                <b-icon pack="mdi" icon="eye" size="is-small"/>
              </b-tag>
              <b-tag type="is-info" v-if="item.file.length > 1">
                <b-icon pack="mdi" icon="file" size="is-small" style="margin-right:0.1em"/>
                {{item.file.length}}
              </b-tag>
              <b-tag type="is-warning" v-if="item.star_rating > 0">
                <b-icon pack="mdi" icon="star" size="is-small"/>
                {{item.star_rating}}
              </b-tag>
            </div>
          </div>
        </div>
      </div>
    </div>

    <div style="padding-top:4px;">

      <watchlist-button :item="item"/>
      <favourite-button :item="item"/>
      <edit-button :item="item" v-if="this.$store.state.optionsWeb.web.sceneEdit" />

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
import { format, parseISO } from 'date-fns'
import WatchlistButton from '../../components/WatchlistButton'
import FavouriteButton from '../../components/FavouriteButton'
import EditButton from '../../components/EditButton'
import StarRating from 'vue-star-rating'

export default {
  name: 'SceneCard',
  props: { item: Object },
  components: { WatchlistButton, FavouriteButton, EditButton, StarRating },
  data () {
    return {
      preview: false,
      format,
      parseISO
    }
  },
  methods: {
    getImageURL (u) {
      if (u.startsWith('http')) {
        return '/img/700x/' + u.replace('://', ':/')
      } else {
        return u
      }
    },
    showDetails (scene) {
      this.$store.commit('overlay/showDetails', { scene: scene })
    }
  }
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

  .overlay {
    flex: 1 0 calc(25%);
    display: flex;
    align-items: center;
    justify-content: center;
    overflow: hidden;
    padding: 0;
    line-height: 0;
    position: absolute;
    left: 0;
    top: 0;
    right: 0;
    bottom: 0;
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
    margin-left: 0.2em;
  }
</style>
