<template>
  <div class="content">
    <div class="columns">
      <div class="column">
        <h3 class="title">{{$t('Mainstream sites')}}</h3>
      </div>
      <div class="column buttons" align="right">
        <a class="button is-primary" v-on:click="taskScrape('_enabled')">{{$t('Run selected scrapers')}}</a>
      </div>
    </div>
    <div class="columns is-multiline">
      <div class="column is-multiline is-one-third" v-for="item in items" :key="item.id">
        <div :class="[runningScrapers.includes(item.id) ? 'card running' : 'card']">
          <div class="card-content content">
            <p class="image is-32x32 is-pulled-left avatar">
              <vue-load-image>
                <img slot="image" :src="getImageURL(item.avatar_url ? item.avatar_url : '/ui/images/blank.png')"/>
                <img slot="preloader" src="/ui/images/blank.png"/>
                <img slot="error" src="/ui/images/blank.png"/>
              </vue-load-image>
            </p>

            <h5 class="title">{{item.name}}</h5>
            <p :class="[runningScrapers.includes(item.id) ? 'invisible' : '']">
              <small v-if="item.last_update !== '0001-01-01T00:00:00Z'">
                Updated {{formatDistanceToNow(parseISO(item.last_update))}} ago</small>
              <small v-else>{{$t('Never scraped')}}</small>
            </p>
            <p :class="[runningScrapers.includes(item.id) ? '' : 'invisible']">
              <small>{{$t('Scraping now...')}}</small>
            </p>
            <div class="switch">
              <b-switch :value="item.is_enabled" @input="$store.dispatch('optionsSites/toggleSite', {id: item.id})"/>
            </div>
            <div class="menu">
              <b-dropdown aria-role="list" class="is-pulled-right" position="is-bottom-left">
                <template slot="trigger">
                  <b-icon icon="dots-vertical"></b-icon>
                </template>
                <b-dropdown-item aria-role="listitem" @click="taskScrape(item.id)">
                  {{$t('Run this scraper')}}
                </b-dropdown-item>
                <b-dropdown-item aria-role="listitem" @click="forceSiteUpdate(item.name)">
                  {{$t('Force update scenes')}}
                </b-dropdown-item>
                <b-dropdown-item aria-role="listitem" @click="deleteScenes(item.name)">
                  {{$t('Delete scraped scenes')}}
                </b-dropdown-item>
              </b-dropdown>
            </div>
          </div>
        </div>
      </div>
    </div>
    <div class="columns is-multiline">
      <div class="column is-multiline is-one-third">
        <h3 class="title">{{$t('JAVR scraper')}}</h3>
        <div class="card">
          <div class="card-content content">
            <h5 class="title">R18</h5>
            <b-field grouped>
              <b-input v-model="javrQuery" placeholder="URL or ID (XXXX-001)" type="search"></b-input>
              <b-button class="button is-primary" v-on:click="scrapeJAVR()">{{$t('Go')}}</b-button>
            </b-field>
          </div>
        </div>
      </div>
      <div class="column is-multiline is-one-third">
        <h3 class="title">{{$t('Custom scene')}}</h3>
        <div class="card">
          <div class="card-content content">
            <b-field label="Scene title" label-position="on-border">
              <b-input v-model="customSceneTitle" placeholder="Stepsis stuck in washing machine" type="search"></b-input>
            </b-field>
            <b-field label="Scene ID" label-position="on-border">
              <b-input v-model="customSceneID" placeholder="Can be empty" type="search"></b-input>
            </b-field>
            <b-field label-position="on-border">
              <b-button class="button is-primary" v-on:click="addScene()">{{$t('Add')}}</b-button>
            </b-field>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import ky from 'ky'
import VueLoadImage from 'vue-load-image'
import { formatDistanceToNow, parseISO } from 'date-fns'

export default {
  name: 'OptionsSites',
  components: { VueLoadImage },
  data () {
    return {
      javrQuery: ''
    }
  },
  mounted () {
    this.$store.dispatch('optionsSites/load')
  },
  methods: {
    getImageURL (u) {
      if (u.startsWith('http')) {
        return '/img/128x/' + u.replace('://', ':/')
      } else {
        return u
      }
    },
    addScene() {
      if (this.customSceneTitle !== '') {
        ky.post('/api/scene/create', { json: { title: this.customSceneTitle, id: this.customSceneID } })
      }
    },
    taskScrape (site) {
      ky.get(`/api/task/scrape?site=${site}`)
    },
    forceSiteUpdate (site) {
      site = this.sanitizeSiteName(site);
      ky.post('/api/options/scraper/force-site-update', {
        json: { site_name: site }
      })
      this.$buefy.toast.open(`Scenes from ${site} will be updated on next scrape`)
    },
    deleteScenes (site) {
      site = this.sanitizeSiteName(site);
      this.$buefy.dialog.confirm({
        title: this.$t('Delete scraped scenes'),
        message: `You're about to delete scraped scenes for <strong>${site}</strong>. Previously matched files will return to unmatched state.`,
        type: 'is-danger',
        hasIcon: true,
        onConfirm: function () {
          ky.post('/api/options/scraper/delete-scenes', {
            json: { site_name: site }
          })
        }
      })
    },
    sanitizeSiteName(site) {
      return site.split('(')[0].trim();
    },
    scrapeJAVR () {
      ky.post('/api/task/scrape-javr', { json: { q: this.javrQuery } })
    },
    parseISO,
    formatDistanceToNow
  },
  computed: {
    items () {
      return this.$store.state.optionsSites.items
    },
    runningScrapers () {
      this.$store.dispatch('optionsSites/load')
      return this.$store.state.messages.runningScrapers
    }
  }
}
</script>

<style scoped>
  .running {
    opacity: 0.6;
    pointer-events: none;
  }

  .card {
    overflow: visible;
    height: 100%;
  }

  .card-content {
    padding-top: 1em;
    padding-left: 1em;
  }

  .avatar {
    margin-right: 1em;
  }

  p {
    margin-bottom: 0.5em !important;
  }

  h5 {
    margin-bottom: 0.25em !important;
  }

  .switch {
    position: absolute;
    bottom: 0.25em;
    right: 0em;
  }

  .invisible {
    display: none;
  }

  .menu {
    position: absolute;
    top: 0.75em;
    right: 0.5em;
  }
</style>
