<template>
  <div class="card is-shadowless">
    <div class="card-image">
      <div class="bbox"
           v-bind:style="{backgroundImage: `url(${getImageURL(item.cover_url)})`, backgroundSize: 'contain', backgroundPosition: 'center', backgroundRepeat: 'no-repeat', opacity:item.is_available ? 1.0 : 0.4}"
           @click="showDetails(item)"
           @mouseover="preview = true"
           @mouseleave="preview = false">
        <video v-if="preview && item.has_preview" :src="`/api/dms/preview/${item.scene_id}`" autoplay loop></video>
        <div class="overlay align-bottom-left">
          <div style="padding: 5px">
            <b-tag v-if="item.is_watched && !this.$store.state.optionsWeb.web.sceneWatched">
              <b-icon pack="mdi" icon="eye" size="is-small"/>
            </b-tag>
            <b-tag type="is-info" v-if="videoFilesCount > 1 && !item.is_multipart">
              <b-icon pack="mdi" icon="file" size="is-small" style="margin-right:0.1em"/>
              {{videoFilesCount}}
            </b-tag>
            <b-tag type="is-info" v-if="item.is_scripted">
              <b-icon pack="mdi" icon="pulse" size="is-small"/>
              <span v-if="scriptFilesCount > 1">{{scriptFilesCount}}</span>
            </b-tag>
            <b-tag type="is-info" v-if="hspFilesCount > 0 && this.$store.state.optionsWeb.web.showHspFile">
              <b-icon pack="mdi" icon="safety-goggles" size="is-small"/>
              <span v-if="hspFilesCount > 1">{{hspFilesCount}}</span>
            </b-tag>
            <b-tag type="is-info" v-if="subtitlesFilesCount > 0 && this.$store.state.optionsWeb.web.showSubtitlesFile">
              <b-icon pack="mdi" icon="subtitles" size="is-small"/>
              <span v-if="subtitlesFilesCount > 1">{{subtitlesFilesCount}}</span>
            </b-tag>
            <b-tag type="is-info" v-if="item.cuepoints != null && item.cuepoints.length > 0 && this.$store.state.optionsWeb.web.sceneCuepoint">
              <b-icon pack="mdi" icon="skip-next-outline" size="is-small"/>
              <span v-if="item.cuepoints != null && item.cuepoints.length > 1">{{item.cuepoints.length}}</span>
            </b-tag>
            <b-tag type="is-warning" v-if="item.star_rating > 0">
              <b-icon pack="mdi" icon="star" size="is-small"/>
              {{item.star_rating}}
            </b-tag>
            <b-tag type="is-info" v-if="item.duration > 0 && this.$store.state.optionsWeb.web.sceneDuration">
              <b-icon pack="mdi" icon="clock" size="is-small"/>
              {{item.duration}}m
            </b-tag>
          </div>
          <div v-if="this.$store.state.optionsWeb.web.showScriptHeatmap && (f = getFunscript())" style="padding: 0px 5px 5px">
            <div v-if="f.has_heatmap" class="heatmapFunscript">
              <img :src="getHeatmapURL(f.id)"/>
            </div>
          </div>
        </div>
      </div>
    </div>

    <div style="padding-top:4px;">
      <div class="scene_title">{{item.title}}</div>

      <hidden-button :item="item" v-if="this.$store.state.optionsWeb.web.sceneHidden"/>
      <watchlist-button :item="item" v-if="this.$store.state.optionsWeb.web.sceneWatchlist"/>
      <trailerlist-button :item="item" v-if="this.$store.state.optionsWeb.web.sceneTrailerlist"/>
      <favourite-button :item="item" v-if="this.$store.state.optionsWeb.web.sceneFavourite"/>
      <wishlist-button v-if="this.$store.state.optionsWeb.web.sceneWishlist && !item.is_available" :item="item"/>
      <watched-button :item="item" v-if="this.$store.state.optionsWeb.web.sceneWatched"/>
      <edit-button :item="item" v-if="this.$store.state.optionsWeb.web.sceneEdit" />

      <span class="is-pulled-right" style="font-size:11px;text-align:right;">
        <a v-if="item.members_url != ''" :href="item.members_url" target="_blank" title="Members Link" rel="noreferrer"><b-icon pack="mdi" icon="link-lock" custom-size="mdi-18px" style="height:0.7rem"/></a>
        <a :href="item.scene_url" :class="{'has-text-white has-background-primary-dark': item.is_subscribed }" target="_blank" rel="noreferrer" style="padding:2px">{{item.site}}</a><br/>
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
import WishlistButton from '../../components/WishlistButton'
import WatchedButton from '../../components/WatchedButton'
import EditButton from '../../components/EditButton'
import TrailerlistButton from '../../components/TrailerlistButton'
import HiddenButton from '../../components/HiddenButton'
import ky from 'ky'

export default {
  name: 'SceneCard',
  props: { item: Object, reRead: Boolean },
  components: { WatchlistButton, FavouriteButton, WishlistButton, WatchedButton, EditButton, TrailerlistButton, HiddenButton },
  data () {
    return {
      preview: false,
      format,
      parseISO
    }
  },
  computed: {
    videoFilesCount () {
      if (this.item.file == null) { return 0 }
      let count = 0
      this.item.file.forEach(obj => {
        if (obj.type === 'video') {
          count = count + 1
        }
      })
      return count
    },
    scriptFilesCount () {
      let count = 0
      if (this.item.file == null) { return 0 }
      this.item.file.forEach(obj => {
        if (obj.type === 'script') {
          count = count + 1
        }
      })
      return count
    },
    hspFilesCount () {
      let count = 0
      if (this.item.file == null) { return 0 }
      this.item.file.forEach(obj => {
        if (obj.type === 'hsp') {
          count = count + 1
        }
      })
      return count
    },
    subtitlesFilesCount () {
      if (this.item.file == null) { return 0 }
      let count = 0
      this.item.file.forEach(obj => {
        if (obj.type === 'subtitles') {
          count = count + 1
        }
      })
      return count
    }
  },
  methods: {
    getImageURL (u) {
      if (u.startsWith('http')) {
        return '/img/700x/' + encodeURI(u)
      } else {
        return u
      }
    },
    showDetails (scene) {
      // reRead is required when the SceneCard is clicked from the ActorDetails
      // the Scenes associated Tables such as Tags, Cast arwon't be Preloaded and
      // will cause errors when the Details Overlay loads
      if (this.reRead) {
        ky.get('/api/scene/'+scene.id).json().then(data => {
          if (data.id != 0){
            this.$store.commit('overlay/showDetails', { scene: data })
          }
        })
      } else {
        this.$store.commit('overlay/showDetails', { scene: scene })
      }
      this.$store.commit('overlay/hideActorDetails')
    },
    getHeatmapURL (fileId) {
      return `/api/dms/heatmap/${fileId}`
    },
    getFunscript () {
      return this.item.file !== null && this.item.file.find(a => a.type === 'script' && a.has_heatmap);
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
    pointer-events: none;
  }

  .align-bottom-left {
    align-items: flex-end;
    justify-content: flex-end;
    flex-wrap: wrap;
    flex-direction: column
  }

  .bbox:after {
    content: '';
    display: block;
    padding-bottom: 100%;
  }

  .tag {
    margin-left: 0.2em;
  }

  .scene_title {
    font-size: 12px;
    text-align: right;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

.heatmapFunscript {
  width: auto;
}

.heatmapFunscript img {
  border: 1px #888 solid;
  width: 100%;
  height: 15px;
  border-radius: 0.25rem;
}

</style>
