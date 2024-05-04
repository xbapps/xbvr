<template>
  <div class="container">
    <b-loading :is-full-page="false" :active.sync="isLoading"></b-loading>
    <div class="content">
      <h3 class="title">{{$t('Import Japanese Adult VR (JAVR) Scene')}}</h3>
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
            <b-button class="button is-primary" v-on:click="addScene(true)" style="margin-left:0.2em">{{$t('Create and Edit')}}</b-button>
          </b-field>
        </div>
      </div>

      <h3 class="title">{{$t('Scrape a scene')}}</h3>
      <div class="card">
        <div class="card-content content">
          <b-field label="Scene URL" label-position="on-border">
            <b-input v-model="scrapeUrl" placeholder="Scene Url - do not use links requiring a login" type="url"></b-input>
          </b-field>
          <b-tooltip :label="$t(`Warning: Ensure you are entering a link to a scene (best taken from viewing the scene). Links to something like a Category or Studio list may result in a corrupt scene you cannot delete. DO NOT USE links requiring logons. Use with caution`)" :delay="50" multilined type="is-danger">
            <b-button class="button is-primary" v-on:click="scrapeSingleScene()">{{$t('Scrape')}}</b-button>
          </b-tooltip>
        </div>
      </div>

      <b-modal :active.sync="isSingleScrapeModalActive"
              has-modal-card
              trap-focus
              aria-role="dialog"
              aria-modal>
        <div class="modal-card" style="width: auto">
          <header class="modal-card-head">
            <p class="modal-card-title">{{$t('Scene Id Required')}}</p>
          </header>
          <section class="modal-card-body">
            <b-field label="Scene Id">
              <b-input
                v-model='singleScrapeId'
                placeholder="eg 12345 (excl site prefix)"
                >
              </b-input>
            </b-field>
          </section>
          <footer class="modal-card-foot">
            <button class="button is-primary" :disabled="this.singleScrapeId == ''" @click="scrapeSingleScene()">Continue</button>
          </footer>
        </div>
      </b-modal>
    </div>
  </div>
</template>

<script>
import ky from 'ky'

export default {
  name: 'OptionsCreateScene',
  data () {
    return {
      isLoading: true,
      javrScraper: 'javdatabase',
      javrQuery: '',
      tpdbSceneUrl: '',
      customSceneTitle: '',
      customSceneID: '',
      scrapeUrl: '',
      isSingleScrapeModalActive: false,
      singleScrapeId: '',
      additionalInfo: [],
    }
  },
  async mounted () {
    await this.loadState()
    this.$store.dispatch('optionsVendor/load')

    if (this.$store.state.optionsSceneCreate.scrapeScene!='') {
      this.scrapeUrl=this.$store.state.optionsSceneCreate.scrapeScene
      this.$store.commit('optionsSceneCreate/setScrapeScene', "")
      this.scrapeSingleScene()
    } 
  },
  methods: {
    async loadState () {
      this.isLoading = true
      await ky.get('/api/options/state')
        .json()
        .then(data => {
          this.javrScraper = data.config.scraper_settings.javr.javrScraper
          this.isLoading = false
        })
    },
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
    scrapeSingleScene () {
      this.additionalInfo = []
      if (this.scrapeUrl.toLowerCase().includes("wetvr.com")) {
        // we need a scene id for wetvr
        if (this.singleScrapeId=="") {
          this.isSingleScrapeModalActive = true
          return
        } else {
          this.isSingleScrapeModalActive = false          
          this.additionalInfo = [{fieldName: "scene_id", fieldPrompt: "Scene Id", placeholder: "eg 69037", fieldValue: this.singleScrapeId, required: true, type: 'number'}]
        }
      }      

      let site = ""
      this.$store.state.optionsVendor.scrapers.forEach((element) => {
        if (this.scrapeUrl.toLowerCase().includes(element.domain)) {
          site = element.id
        }
      });
      if (this.scrapeUrl.toLowerCase().includes("sexlikereal.com")) {
        site = "slr-single_scene"
      }
      if (this.scrapeUrl.toLowerCase().includes("czechvrnetwork.com")) {
        site = "czechvr-single_scene"
      }
      if (this.scrapeUrl.toLowerCase().includes("povr.com")) {
        site = "povr-single_scene"
      }
      if (this.scrapeUrl.toLowerCase().includes("vrporn.com")) {
        site = "vrporn-single_scene"
      }
      if (this.scrapeUrl.toLowerCase().includes("vrphub.com")) {
        site = "vrphub-single_scene"
      }
      if (site == "") {
        this.$buefy.toast.open({message: `No scrapers exist for this domain`, type: 'is-danger', duration: 5000})      
        return
      }    
      
      switch (site) {
            case "wetvr":
            case "sexbabesvr":
            case "tonightsgirlfriend":
              var fieldCheckMsg="Please check the Release Date"
              break
            case "fuckpassvr-native":
              var fieldCheckMsg="Note: Video Previews are not available when scraping single scenes from FuckpassVR"
              break
            case "lethalhardcorevr":
              var fieldCheckMsg=`Please check the Site if the scene was for WhorecraftVR. Please check the Release Date`
              break
            case "littlecaprice":
              var fieldCheckMsg=`Please specify a URL for the cover image`
              break            
            case "stasyqvr":
              var fieldCheckMsg=`Please specify a Duration if required`
              break
            case "bvr ":
              var fieldCheckMsg=`Please check the Release Date and specify a Duration if required`
              break
            default:
                var fieldCheckMsg=""                
          }

      if (fieldCheckMsg != "") {
        this.$buefy.toast.open({message: `Scene scraping in progress, please wait for the Scene Detail popup`, type: 'is-warning', duration: 5000})
      } else {
        this.$buefy.toast.open({message: `Scene scraping in progress`, type: 'is-warning', duration: 5000})
      }
      ky.post(`/api/task/singlescrape`, {timeout: false, json: { site: site, sceneurl: this.scrapeUrl, additionalinfo: this.additionalInfo}})
      .json()
      .then(data => { 
        if (data.status == 'OK') {          
          this.$store.commit('overlay/editDetails', { scene: data.scene })
          if (fieldCheckMsg != "") {
            this.$buefy.toast.open({message: fieldCheckMsg, type: 'is-warning', duration: 10000})
          }
        }
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
