<template>
  <div class="content">
    <div class="columns">
      <div class="column">
        <h3 class="title">{{$t('Scrape scenes from studios')}}</h3>
      </div>
      <div class="column buttons" align="right">
        <a class="button is-primary" v-on:click="taskScrape('_enabled')">{{$t('Run selected scrapers')}}</a>
      </div>
    </div>
    <b-table :data="scraperList">
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
      <b-table-column field="options" :label="opt" v-slot="props" width="30">
        <div class="menu">
          <b-dropdown aria-role="list" class="is-pulled-right" position="is-bottom-left">
            <template slot="trigger">
              <b-icon icon="dots-vertical mdi-18px"></b-icon>
            </template>
            <b-dropdown-item aria-role="listitem" @click="taskScrape(props.row.id)">
              {{$t('Run this scraper')}}
            </b-dropdown-item>
            <b-dropdown-item aria-role="listitem" @click="forceSiteUpdate(props.row.name)">
              {{$t('Force update scenes')}}
            </b-dropdown-item>
            <b-dropdown-item aria-role="listitem" @click="deleteScenes(props.row.name)">
              {{$t('Delete scraped scenes')}}
            </b-dropdown-item>
          </b-dropdown>
        </div>
      </b-table-column>
    </b-table>
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
      tpdbSceneUrl: ''
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
      if (site.includes('(Custom ')) {
        return site.replace('(Custom ','(') 
      }      
      return site.split('(')[0].trim();
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
