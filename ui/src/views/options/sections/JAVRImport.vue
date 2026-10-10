<template>
  <div class="card">
    <div class="card-content content">
      <b-field grouped>
        <b-select placeholder="Select scraper" v-model="javrScraper">
          <option value="javdatabase">javdatabase.com</option>
          <option value="r18d">r18.dev</option>
          <option value="javlibrary">javlibrary.com</option>
          <option value="javland">jav.land</option>
        </b-select>
        <b-input v-model="javrQuery" placeholder="ID (xxxx-001)" type="search"></b-input>
        <b-button class="button is-primary" v-on:click="scrapeJAVR()">{{$t('Go')}}</b-button>
      </b-field>
      <p style="font-size: 0.75em;">
        <span style="color: red; font-weight: bold;">R18.dev:</span>
        DVD-IDs like <span style="font-family: monospace;color:#7957d5">VRKM-139</span> will only work for scenes that were released on r18.com prior to 2022-06.
        <br>
        All newer scenes should be scraped using their full FANZA content ID, e.g. <span style="font-family: monospace;color:#7957d5">84vrkm00139</span> or using one of the other scrapers.
      </p>
    </div>
  </div>
</template>

<script>
import ky from 'ky'

export default {
  name: 'JAVRImport',
  data () {
    return {
      javrScraper: 'javdatabase',
      javrQuery: ''
    }
  },
  async mounted () {
    await ky.get('/api/options/state')
      .json()
      .then(data => {
        this.javrScraper = data.config.scraper_settings.javr.javrScraper
      })
  },
  methods: {
    scrapeJAVR () {
      ky.post('/api/task/scrape-javr', { json: { s: this.javrScraper, q: this.javrQuery } })
    }
  }
}
</script>

<style scoped>
  .card {
    overflow: visible;
    height: 100%;
  }

  .card-content {
    padding-top: 1em;
    padding-left: 1em;
  }
</style>
