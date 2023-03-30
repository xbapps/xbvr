<template>
  <div class="content">
    <b-loading :is-full-page="true" :active.sync="isLoading"></b-loading>
    <div class="columns">
      <div class="column">
        <h3 class="title">{{$t('Scrape scenes from studios')}}</h3>
      </div>
      <div class="column buttons" align="right">
        <a class="button is-primary" v-on:click="taskScrape('_enabled')">{{$t('Run selected scrapers')}}</a>
      </div>
    </div>
    <b-table :data="scraperList" ref="scraperTable">
      <b-table-column field="is_enabled" :label="$t('Enabled')" v-slot="props" width="60" sortable>
          <span><b-switch v-model ="props.row.is_enabled" @input="$store.dispatch('optionsSites/toggleSite', {id: props.row.id})"/></span>
      </b-table-column>
      <b-table-column field="icon" width="50" v-slot="props" cell-class="narrow">
            <span class="image is-32x32">
              <vue-load-image>
                <img slot="image" :src="getImageURL(props.row.avatar_url ? props.row.avatar_url : '/ui/images/blank.png')"/>
                <img slot="preloader" src="/ui/images/blank.png"/>
                <img slot="error" src="/ui/images/blank.png"/>
              </vue-load-image>
            </span>
      </b-table-column>
      <b-table-column field="sitename" :label="$t('Studio')" sortable searchable v-slot="props">
        {{ props.row.sitename }}
      </b-table-column>
      <b-table-column field="source" :label="$t('Source')" sortable searchable v-slot="props">
        {{ props.row.source }}
      </b-table-column>
      <b-table-column field="last_update" :label="$t('Last scrape')" sortable v-slot="props">
            <span :class="[runningScrapers.includes(props.row.id) ? 'invisible' : '']">
              <span v-if="props.row.last_update !== '0001-01-01T00:00:00Z'">
                {{formatDistanceToNow(parseISO(props.row.last_update))}} ago</span>
              <span v-else>{{$t('Never scraped')}}</span>
            </span>
            <span :class="[runningScrapers.includes(props.row.id) ? '' : 'invisible']">
              <span class="pulsate is-info">{{$t('Scraping now...')}}</span>
            </span>
      </b-table-column>
      <b-table-column field="subscribed" :label="$t('Subscribed')" v-slot="props" width="60" sortable>
          <span><b-switch v-model ="props.row.subscribed" @input="$store.dispatch('optionsSites/toggleSubscribed', {id: props.row.id})"/></span>
      </b-table-column>
      <b-table-column field="options" v-slot="props" width="30">
        <div class="menu">
          <b-dropdown aria-role="list" class="is-pulled-right" position="is-bottom-left">
            <template slot="trigger">
              <b-icon icon="dots-vertical mdi-18px"></b-icon>
            </template>
            <b-dropdown-item aria-role="listitem" @click="taskScrape(props.row.id)">
              {{$t('Run this scraper')}}
            </b-dropdown-item>
            <b-dropdown-item aria-role="listitem" @click="forceSiteUpdate(props.row.name, props.row.id)">
              {{$t('Force update scenes')}}
            </b-dropdown-item>
            <b-dropdown-item aria-role="listitem" @click="deleteScenes(props.row.name, props.row.id)">
              {{$t('Delete scraped scenes')}}
            </b-dropdown-item>
          </b-dropdown>
        </div>
      </b-table-column>
    </b-table>
    <div class="columns">
      <div class="column">
      </div>
        <div class="column buttons" align="right">
          <a class="button is-small" v-on:click="toggleAllSubscriptions()">{{$t('Toggle Subscriptions of all visible sites')}}</a>
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
      javrQuery: '',
      tpdbSceneUrl: '',
      isLoading: false
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
    taskScrape (scraper) {
      ky.get(`/api/task/scrape?site=${scraper}`)
    },
    forceSiteUpdate (site, scraper) {
      ky.post('/api/options/scraper/force-site-update', {
        json: { scraper_id: scraper }
      })
      this.$buefy.toast.open(`Scenes from ${site} will be updated on next scrape`)
    },
    deleteScenes (site, scraper) {
      this.$buefy.dialog.confirm({
        title: this.$t('Delete scraped scenes'),
        message: `You're about to delete scraped scenes for <strong>${site}</strong>. Previously matched files will return to unmatched state.`,
        type: 'is-danger',
        hasIcon: true,
        onConfirm: function () {
          ky.post('/api/options/scraper/delete-scenes', {
            json: { scraper_id: scraper }
          })
        }
      })
    },
    async toggleAllSubscriptions(){
      const table = this.$refs.scraperTable;
      this.isLoading=true
      for (let i=0; i<table.newData.length; i++) {
        await ky.put(`/api/options/sites/subscribed/${table.newData[i].id}`, { json: {} }).json()
        this.$store.dispatch('optionsSites/load')
      }
      this.isLoading=false
    },
    parseISO,
    formatDistanceToNow
  },
  computed: {
    scraperList() {
      var items = this.$store.state.optionsSites.items;
      let re = /(.*)\s+\((.+)\)$/;
      for (let i=0; i < items.length; i++) {
        items[i].sitename = items[i].name;
        items[i].source = "";

        var m = re.exec(items[i].name);
        if (m) {
          items[i].sitename = m[1];
          items[i].source = m[2];
        }
      }
      return items;
    },
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

  .invisible {
    display: none;
  }
  .pulsate {
    -webkit-animation: pulsate 0.8s linear;
    -webkit-animation-iteration-count: infinite;
    opacity: 0.5;
  }

  @-webkit-keyframes pulsate {
    0% {
      opacity: 0.5;
    }
    50% {
      opacity: 1.0;
    }
    100% {
      opacity: 0.5;
    }
  }
</style>

<style>
  .content table td.narrow{
    padding-top: 5px;
    padding-bottom: 2px;
  }
</style>
