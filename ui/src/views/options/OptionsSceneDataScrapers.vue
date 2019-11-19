<template>
  <div class="content">
    <h3 class="title">Mainstream scrapers</h3>
    <div class="buttons">
      <a class="button is-primary" v-on:click="taskScrape('_enabled')">Run Selected Scrapers</a>
    </div>

    <div class="columns is-multiline">
      <div class="column is-multiline is-one-third"
           v-for="item in items" :key="item.id">
        <div :class="[runningScrapers.includes(item.id) ? 'card pulsate' : 'card']">
          <div class="card-content content">
            <h5 class="title">{{item.name}}</h5>
            <p>
              <small v-if="item.last_update !== '0001-01-01T00:00:00Z'">
                Updated {{formatDistanceToNow(parseISO(item.last_update))}} ago
              </small>
              <small v-else>Never scraped</small>
            </p>

            <div class="switch">
              <b-switch :value="item.is_enabled"
                        @input="$store.dispatch('optionsSites/toggleSite', {id: item.id})"/>
            </div>

            <div class="menu">
              <b-dropdown aria-role="list" class="is-pulled-right" position="is-bottom-left">
                <template slot="trigger">
                  <b-icon icon="dots-vertical"></b-icon>
                </template>
                <b-dropdown-item aria-role="listitem" @click="taskScrape(item.id)">Scrape this site</b-dropdown-item>
                <b-dropdown-item aria-role="listitem" @click="forceSiteUpdate(item.name)">Force update scenes
                </b-dropdown-item>
              </b-dropdown>
            </div>
          </div>
        </div>
      </div>
    </div>

  </div>
</template>

<script>
  import ky from "ky";
  import {formatDistanceToNow, parseISO} from "date-fns";

  export default {
    name: "OptionsSites",
    mounted() {
      this.$store.dispatch("optionsSites/load");
    },
    methods: {
      taskScrape(site) {
        ky.get(`/api/task/scrape?site=${site}`);
      },
      forceSiteUpdate(site) {
        ky.post(`/api/config/scraper/force-site-update`, {
          json: {"site_name": site}
        });
        this.$buefy.toast.open(`Scenes from ${site} will be updated on next scrape`);
      },
      parseISO,
      formatDistanceToNow,
    },
    computed: {
      items() {
        return this.$store.state.optionsSites.items;
      },
      runningScrapers() {
        this.$store.dispatch("optionsSites/load");
        return this.$store.state.messages.runningScrapers;
      }
    }
  }
</script>

<style scoped>
  .pulsate {
    -webkit-animation: pulsate 0.5s linear;
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

  .card {
    height: 100%;
  }

  .card-content {
    padding-top: 1em;
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

  .menu {
    position: absolute;
    top: 0.75em;
    right: 0.5em;
  }
</style>
