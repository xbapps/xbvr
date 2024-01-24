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
        <b-tooltip class="is-warning" :active="props.row.has_scraper == false" :label="$t('Scraper does not exist')"  :delay="250" >
          <span :class="[props.row.has_scraper ? '' : 'has-text-danger']">{{ props.row.sitename }}</span>
        </b-tooltip>
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
      <b-table-column field="limit_scraping" :label="$t('Limit Scraping')" v-slot="props" width="60" sortable>
        <b-tooltip class="is-info" :label="$t('Limit scraping to newest scenes on the website. Turn off if you are missing scenes.')" :delay="250" >
          <span><b-switch v-model ="props.row.limit_scraping" @input="$store.dispatch('optionsSites/toggleLimitScraping', {id: props.row.id})"/></span>
        </b-tooltip>
      </b-table-column>
      <b-table-column field="subscribed" :label="$t('Subscribed')" v-slot="props" width="60" sortable>
        <b-tooltip class="is-info" :label="$t('Highlights this studio in the scene view and includes scenes in the &quot;Has subscription&quot; attribute filter')" :delay="250" >
          <span v-if="props.row.master_site_id==''"><b-switch v-model ="props.row.subscribed" @input="$store.dispatch('optionsSites/toggleSubscribed', {id: props.row.id})"/></span>
        </b-tooltip>
      </b-table-column>
      <b-table-column field="options" v-slot="props" width="30">
        <div class="menu">
          <b-dropdown aria-role="list" class="is-pulled-right" position="is-bottom-left">
            <template slot="trigger">
              <b-icon icon="dots-vertical mdi-18px"></b-icon>
            </template>
            <b-dropdown-item v-if="props.row.has_scraper" aria-role="listitem" @click="taskScrape(props.row.id)">
              {{$t('Run this scraper')}}
            </b-dropdown-item>
            <b-dropdown-item v-if="props.row.has_scraper && props.row.id != 'baberoticavr'" aria-role="listitem" @click="taskScrapeScene(props.row.id)">
              {{$t('Scrape Single Scene')}}
            </b-dropdown-item>
            <b-dropdown-item v-if="props.row.has_scraper && props.row.master_site_id==''" aria-role="listitem" @click="forceSiteUpdate(props.row.name, props.row.id)">
              {{$t('Force update scenes')}}
            </b-dropdown-item>
            <b-dropdown-item v-if="props.row.has_scraper && props.row.master_site_id!=''" aria-role="listitem" @click="removeSceneLinks(props.row, true)">
              {{$t('Remove Scene Links')}}
            </b-dropdown-item>
            <b-dropdown-item v-if="props.row.has_scraper && props.row.master_site_id!=''" aria-role="listitem" @click="removeSceneLinks(props.row, false)">
              {{$t('Remove Scene Links (Keep edits)')}}
            </b-dropdown-item>
            <b-dropdown-item aria-role="listitem" @click="deleteScenes(props.row)">
              {{$t('Delete scraped scenes')}}
            </b-dropdown-item>
            <b-dropdown-item aria-role="listitem" @click="scrapeActors(props.row.name, props.row.id)" v-if="props.row.master_site_id==''">
              {{$t('Scrape Actor Details from Site')}}
            </b-dropdown-item>
          </b-dropdown>
        </div>
      </b-table-column>
      <b-table-column field="master_site_id" :label="$t('Main Site')" v-slot="props" width="60" sortable>
        <span>
          <a @click="editMatchParams(props.row)" title="Edit Scene Matching Parameters" v-if="props.row.master_site_id != ''"> 
            <b-icon pack="mdi" icon="cog-outline" size="is-small"/>
          </a>
          {{getMasterSiteName(props.row.master_site_id)}}
        </span>
      </b-table-column>
    </b-table>
    <div class="columns">
      <div class="column">
      </div>
        <div class="column buttons" align="right">
          <a class="button is-small" v-on:click="toggleAllLimitScraping()">{{$t('Toggle Limit Scraping of all visible sites')}}</a>
          <a class="button is-small" v-on:click="toggleAllSubscriptions()">{{$t('Toggle Subscriptions of all visible sites')}}</a>
        </div>
    </div>

    <b-modal :active.sync="isSingleScrapeModalActive"
             has-modal-card
             trap-focus
             aria-role="dialog"
             aria-modal>
      <div class="modal-card" style="width: auto">
        <header class="modal-card-head">
          <p class="modal-card-title">{{$t('Additional Details Required')}}</p>
        </header>
        <section class="modal-card-body">
          <b-field v-if="additionalInfoIdx == 0 && this.scraperwarning != ''"><span>{{this.scraperwarning}}</span></b-field>
          <b-field v-if="additionalInfoIdx == 0 && this.scraperwarning2 != ''"><span>{{this.scraperwarning2}}</span></b-field>          
          <b-field :label=this.additionalInfo[additionalInfoIdx].fieldPrompt>
            <b-input v-if="additionalInfo[additionalInfoIdx].type != 'checkbox'"
              :type=additionalInfo[additionalInfoIdx].type
              v-model='additionalInfo[additionalInfoIdx].fieldValue'
              :required=additionalInfo[additionalInfoIdx].required
              :placeholder=additionalInfo[additionalInfoIdx].placeholder                            
              ref="additionInfoInput"
              >
            </b-input>
            <b-checkbox v-if="additionalInfo[additionalInfoIdx].type == 'checkbox'" v-model="additionalInfo[additionalInfoIdx].fieldValue">{{this.additionalInfo[additionalInfoIdx].fieldPrompt}}</b-checkbox>
          </b-field>
        </section>
        <footer class="modal-card-foot">
          <button class="button is-primary" :disabled="this.additionalInfo[additionalInfoIdx].required && this.additionalInfo[additionalInfoIdx].fieldValue == ''" @click="taskScrapeSceneInfoEntered()">Continue
          </button>
        </footer>
      </div>
    </b-modal>

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
      isLoading: false,
      sceneUrl: '',
      isSingleScrapeModalActive: false,
      additionalInfo: [{fieldName: "scene_url", fieldPrompt: "Scene Url", placeholder: "eg https://www.mysite.com/scenes/my scene", fieldValue: '', required: true, type: 'url' }],
      additionalInfoIdx: 0,
      currentScraper: '',
      scraperwarning: '',
      scraperwarning2: '',
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
    taskScrapeScene (scraper) {
      this.currentScraper=scraper      
      this.additionalInfo = [{fieldName: "scene_url", fieldPrompt: "Scene Url", placeholder: "Enter the url for a VR Scene", fieldValue: '', required: true, type: 'url'}]      
      this.scraperwarning = "Take care to only use scene urls for the " + scraper + " Scraper"
      this.scraperwarning2 = ""
      switch (scraper) {
        case 'wankzvr':
        case 'milfvr':
        case 'herpovr':
        case  'brasilvr':
        case 'tranzvr':
          this.scraperwarning = "Only use povr.com urls for the " + scraper + " Scraper"
          break
        case 'tonightsgirlfriend':
          this.scraperwarning2 = "Warning " + scraper + " also includes 2d scenes, only select scenes from their VR section"
        case 'naughtyamericavr':
          this.scraperwarning2 = "Warning The NaughtyAmerica site also includes 2d scenes, only select scenes from their VR section"
          break
    }
      this.additionalInfoIdx=0
      this.isSingleScrapeModalActive = true      
    },
    taskScrapeSceneInfoEntered () {      
      const inputElement = this.$refs.additionInfoInput
      if (!inputElement.isValid) {
        // get the field again
        this.isSingleScrapeModalActive = true
        return
      }

      this.isSingleScrapeModalActive = false      
      var fieldCheckMsg = ""
      if (this.additionalInfo[0].fieldValue.toLowerCase().includes('fuckpassvr.com')) {
        var fieldCheckMsg="Note: Video Previews are not available when scraping single scenes from FuckpassVR"
      }
      if (this.additionalInfo[0].fieldValue.toLowerCase().includes('lethalhardcorevr.com')) {
        var fieldCheckMsg=`Please check the Site if the scene was for WhorecraftVR. Please check the Release Date`
      }
      if (this.additionalInfo[0].fieldValue.toLowerCase().includes('littlecaprice-dreams.com')) {
        var fieldCheckMsg=`Please specify a URL for the cover image`
      }
      if (this.additionalInfo[0].fieldValue.toLowerCase().includes('sexbabesvr.com')) {
        var fieldCheckMsg="Please check the Release Date"
      }
      if (this.additionalInfo[0].fieldValue.toLowerCase().includes('stasyqvr.com')) {
        var fieldCheckMsg=`Please specify a Duration if required`
      }
      if (this.additionalInfo[0].fieldValue.toLowerCase().includes('tonightsgirlfriend.com')) {
        var fieldCheckMsg="Please check the Release Date"
      }
      if (this.additionalInfo[0].fieldValue.toLowerCase().includes('virtualporn.com')) {
        var fieldCheckMsg=`Please check the Release Date and specify a Duration if required`
      }
      if (this.additionalInfo[0].fieldValue.toLowerCase().includes('wetvr.com')) {        
        var fieldCheckMsg="Please check the Release Date"
      }

      if (this.additionalInfoIdx == 0) {
        if (this.additionalInfo[0].fieldValue.toLowerCase().includes('wetvr.com')) {
          this.additionalInfo.push({fieldName: "scene_id", fieldPrompt: "Scene Id", placeholder: "eg 69037 (excl site prefix)", fieldValue: '', required: true, type: 'number'})
        }
      }
      
      this.additionalInfo[this.additionalInfoIdx].fieldValue = this.additionalInfo[this.additionalInfoIdx].fieldValue.trim()
      if (this.additionalInfoIdx + 1 < this.additionalInfo.length) {          
        this.additionalInfoIdx = this.additionalInfoIdx +1
        this.isSingleScrapeModalActive = true      
      } else {
        if (fieldCheckMsg != "") {
          this.$buefy.toast.open({message: `Scene scraping in progress, please wait for the Scene Detail popup`, type: 'is-warning', duration: 5000})
        } else {
          this.$buefy.toast.open({message: `Scene scraping in progress`, type: 'is-warning', duration: 5000})
        }
        ky.post(`/api/task/singlescrape`, {timeout: false, json: { site: this.currentScraper, sceneurl: this.additionalInfo[0].fieldValue, additionalinfo: this.additionalInfo.slice(1)}})
        .json()
        .then(data => { 
          if (data.status == 'OK') {          
            this.$store.commit('overlay/editDetails', { scene: data.scene })
            if (fieldCheckMsg != "") {
              this.$buefy.toast.open({message: fieldCheckMsg, type: 'is-warning', duration: 10000})
            }
          }
        })
      }
    },
    forceSiteUpdate (site, scraper) {
      ky.post('/api/options/scraper/force-site-update', {
        json: { scraper_id: scraper }
      })
      this.$buefy.toast.open(`Scenes from ${site} will be updated on next scrape`)
    },
    deleteScenes (site) {
      this.$buefy.dialog.confirm({
        title: this.$t('Delete scraped scenes'),
        message: `You're about to delete scraped scenes for <strong>${site.name}</strong>.`,
        type: 'is-danger',
        hasIcon: true,
        onConfirm: function () {
          if (site.master_site_id==""){
            ky.post('/api/options/scraper/delete-scenes', {
              json: { scraper_id: site.id }
            })
          } else {
            const external_source = 'alternate scene ' + site.id
            ky.delete(`/api/extref/delete_extref_source`, {
              json: {external_source: external_source}
            });
          }
        }
      })
    },
    removeSceneLinks (site, all) {
      this.$buefy.dialog.confirm({
        title: this.$t('Remove Scene Links'),
        message: `You're about to remove links for scenes from <strong>${site.name}</strong>. Scenes will be relinked after the next scrape.`,
        type: 'is-warning',
        hasIcon: true,
        onConfirm: function () {
          const external_source = 'alternate scene ' + site.id          
          if (all) {
            ky.delete(`/api/extref/delete_extref_source_links/all`, {
              json: {external_source: external_source}
            });
          } else {
            ky.delete(`/api/extref/delete_extref_source_links/keep_manual`, {
              json: {external_source: external_source}
            });
          }
        }
      })
    },
    scrapeActors(site, scraper) {      
      ky.get('/api/extref/generic/scrape_by_site/' + scraper)
      this.$buefy.toast.open(`Scraping Actor Details from ${site}`)
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
    async toggleAllLimitScraping(){
      const table = this.$refs.scraperTable;
      this.isLoading=true
      for (let i=0; i<table.newData.length; i++) {
        await ky.put(`/api/options/sites/limit_scraping/${table.newData[i].id}`, { json: {} }).json()
        this.$store.dispatch('optionsSites/load')
      }
      this.isLoading=false
    },    
    editMatchParams(site){
      this.$store.commit('overlay/showSceneMatchParams', { site: site })
    },
    getMasterSiteName(siteId){
      if (siteId=="") {
        return ""
      }      
      return  this.scraperList.find(element => element.id === siteId).name;
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
