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
            <b-tag type="is-warning" v-if="item.star_rating > 0">
              <b-icon pack="mdi" icon="star" size="is-small"/>
              {{item.star_rating}}
            </b-tag>
          </div>
        </div>
      </div>
    </div>

    <div style="padding-top:4px;">
      <div class="scene_title">{{item.title}}</div>

      <watchlist-button :item="item" v-if="this.$store.state.optionsWeb.web.sceneWatchlist"/>
      <favourite-button :item="item" v-if="this.$store.state.optionsWeb.web.sceneFavourite"/>
      <watched-button :item="item" v-if="this.$store.state.optionsWeb.web.sceneWatched"/>
      <edit-button :item="item" v-if="this.$store.state.optionsWeb.web.sceneEdit" />

      <span class="is-pulled-right" style="font-size:11px;text-align:right;">
        <a :href="item.scene_url" target="_blank" rel="noreferrer">{{item.site}}</a><br/>
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
import WatchedButton from '../../components/WatchedButton'
import EditButton from '../../components/EditButton'

export default {
  name: 'SceneCard',
  props: { item: Object },
  components: { WatchlistButton, FavouriteButton, WatchedButton, EditButton },
  data () {
    return {
      preview: false,
      format,
      parseISO
    }
  },
  computed: {
    videoFilesCount () {
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
      this.item.file.forEach(obj => {
        if (obj.type === 'script') {
          count = count + 1
        }
      })
      return count
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
</style>
