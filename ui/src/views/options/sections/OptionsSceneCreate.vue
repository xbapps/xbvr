<template>
  <div class="content">
    <h3 class="title">{{$t('Import Japanese Adult VR (JAVR) Scene')}}</h3>
    <div class="card">
      <div class="card-content content">
        <b-field grouped>
          <b-select placeholder="Select scraper" v-model="javrScraper">
            <option value="javdatabase">javdatabase.com</option>
            <option value="javlibrary">javlibrary.com</option>
            <option value="javbus">javbus.com</option>
            <option value="javland">jav.land</option>
          </b-select>
          <b-input v-model="javrQuery" placeholder="ID (xxxx-001)" type="search"></b-input>
          <b-button class="button is-primary" v-on:click="scrapeJAVR()">{{$t('Go')}}</b-button>
        </b-field>
      </div>
    </div>

    <h3 class="title">{{$t('Import scene from TPDB')}}</h3>
    <div class="card">
      <div class="card-content content">
        <h5 class="title">API Token</h5>
        <b-field label="TPDB API Token" label-position="on-border" grouped>
          <b-input v-model="tpdbApiToken" placeholder="TPDB API Token" type="search"></b-input>
        </b-field>
        <br>
        <b-field label="TPDB Scene URL" label-position="on-border" grouped>
          <b-input v-model="tpdbSceneUrl" placeholder="TPDB URL" type="search"></b-input>
          <b-button class="button is-primary" v-on:click="scrapeTPDB()">{{$t('Go')}}</b-button>
        </b-field>
      </div>
    </div>

    <h3 class="title">{{$t('Create custom scene')}}</h3>
    <div class="card">
      <div class="card-content content">
        <b-field label="Scene title" label-position="on-border">
          <b-input v-model="customSceneTitle" placeholder="Stepsis stuck in washing machine" type="search"></b-input>
        </b-field>
        <b-field label="Scene ID" label-position="on-border" grouped>
          <b-input v-model="customSceneID" placeholder="Can be empty" type="search"></b-input>
          <b-button class="button is-primary" v-on:click="addScene(false)">{{$t('Create')}}</b-button>
          <b-button class="button is-primary" v-on:click="addScene(true)" style="margin-left:0.2em">{{$t('Create/Edit')}}</b-button>
        </b-field>
      </div>
    </div>
  </div>
</template>

<script>
import ky from 'ky'

export default {
  name: 'OptionsCreateScene',
  data () {
    return {
      javrScraper: 'javdatabase',
      javrQuery: '',
      tpdbSceneUrl: '',
      customSceneTitle: '',
      customSceneID: ''
    }
  },
  mounted () {
    this.$store.dispatch('optionsVendor/load')
  },
  methods: {
    addScene(showEdit) {
      if (this.customSceneTitle !== '') {
        ky.post('/api/scene/create', { json: { title: this.customSceneTitle, id: this.customSceneID } })
        .json()
        .then(scene => {          
          if (showEdit) {
            this.$store.commit('overlay/editDetails', { scene: scene })
          }
        })          
      }
    },
    scrapeJAVR () {
      ky.post('/api/task/scrape-javr', { json: { s: this.javrScraper, q: this.javrQuery } })
    },
    scrapeTPDB () {
      ky.post('/api/task/scrape-tpdb', {
        json: { apiToken: this.tpdbApiToken, sceneUrl: this.tpdbSceneUrl }
      })
    },
  },
  computed: {
    tpdbApiToken: {
      get () {
        return this.$store.state.optionsVendor.tpdb.apiToken
      },
      set (value) {
        this.$store.state.optionsVendor.tpdb.apiToken = value
      }
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

<style>
  .content table td.narrow{
    padding-top: 5px;
    padding-bottom: 2px;
  }
</style>
