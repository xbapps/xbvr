<template>
  <div class="container">
    <b-loading :is-full-page="false" :active.sync="isLoading" />
    <div class="content">
      <h3>{{ $t('Advanced') }}</h3>
      <hr />
      <b-tabs v-model="activeTab" size="medium" type="is-boxed" style="margin-left: 0px" id="importexporttab">
            <b-tab-item label="Scene Details"/>
            <b-tab-item label="Actor Settings"/>
            <b-tab-item label="Create Custom Site"/>
            <b-tab-item :label="$t('Alternate Sites')"/>
      </b-tabs>

      <!-- Screen Details Tab -->
      <div class="columns" v-if="activeTab == 0">
        <div class="column">
          <section>
            <b-field>
              <b-switch v-model="showInternalSceneId" type="is-default">
                show Internal Scene Id
              </b-switch>
            </b-field>
            <b-field>
              <b-switch v-model="showHSPApiLink" type="is-default">
                show Heresphere Api Link
              </b-switch>
            </b-field>
            <b-field>
              <b-tooltip :label="$t('Only required when troubleshooting search issues, this will enable a Tab in the Scene Details to display what search fields exist and their values for a scene')" :delay="500" type="is-warning">
              <b-switch v-model="showSceneSearchField" type="is-default">
                show Scene Search Fields
              </b-switch>
              </b-tooltip>
            </b-field>
            <b-field>
              <b-button type="is-primary" @click="save">Save</b-button>
            </b-field>
          </section>
        </div>
      </div>
      
      <!-- Actor Related Settings -->
      <div class="columns" v-if="activeTab == 1">
        <div class="column">
          <section>
              <b-tooltip :label="$t('Allows the entry of Actor Height/Weight in Imperial measurements')" :delay="500" type="is-warning">
                <b-switch v-model="useImperialEntry" type="is-default">
                  {{ $t('Entry Height/Weight in Imperial Measurements') }}
                </b-switch>
              </b-tooltip>
            <b-field>
              <b-tooltip :label="$t('Scrape Actor details from sites after running a Scene Scrape, otherwise run manually')" :delay="500" type="is-warning">
                <b-switch v-model="scrapeActorAfterScene" type="is-default">
                  {{ $t('Scrape Site Actors after Scene Scrape') }}
                </b-switch>
              </b-tooltip>
            </b-field>
            <b-field :label="$t('Stashdb Api Key')" label-position="on-border">
              <b-input v-model="stashApiKey" placeholder="Visit https://discord.com/invite/2TsNFKt to sign up to Stashdb" type="password"></b-input>
            </b-field>
            <b-field>
              <b-tooltip :active="stashApiKey==''" :label="$t('Enter a StashApi key to enable')" >
                <b-button type="is-primary" :disabled="stashApiKey==''" @click="stashdb">{{ $t('Scrape StashDB') }}</b-button>
              </b-tooltip>
            </b-field>
            
            <b-field :label="$t('DMM Api id')" label-position="on-border">
              <b-input v-model="dmmApiId" placeholder="Visit https://affiliate.dmm.com/api/ to sign up to DMM-api service" type="password"></b-input>
            </b-field>
            <b-field :label="$t('DMM Affiliate id')" label-position="on-border">
              <b-input v-model="dmmAffiliateId" placeholder="Visit https://affiliate.dmm.com/api/ to sign up to DMM-api service" type="password"></b-input>
            </b-field>

            <b-field>
              <b-button type="is-primary" @click="scrapeXbvrActors">{{ $t('Scrape Actor Details from XBVR Sites') }}</b-button>
            </b-field>
            <b-field>
              <b-button type="is-primary" @click="save">Save</b-button>
            </b-field>
          </section>
        </div>
      </div>

      <!-- Custom Sites Tab -->
      <div class="columns" v-if="activeTab == 2">
        <div class="column">
          <section>
            <b-field :label="$t('Scraper Url')" label-position="on-border">
              <b-input v-model="scraperUrl" :placeholder="$t('Enter the Url to Studio Scene List')" @input="validateScraperFields()"></b-input>
            </b-field>
            <b-field :label="$t('Name')" label-position="on-border">
              <b-input v-model="scraperName" :placeholder="$t('Enter Studio Name')" @input="validateScraperFields()"></b-input>
            </b-field>
            <b-field :label="$t('Company')" label-position="on-border">
              <b-input v-model="scraperCompany" :placeholder="$t('Optional: defaults to Name')"></b-input>
            </b-field>
            <b-field :label="$t('Avatar Url')" label-position="on-border">
              <b-input v-model="scraperAvatar" :placeholder="$t('Optional')"></b-input>
            </b-field>  
            <b-field :label="$t('Main Site')" label-position="on-border" :addons="true" class="field-extra">
              <b-tooltip :label="$t('Leave blank, unless you want to link scenes from the new custom site to scenes on an existing studio site, e.g. VRHush on SLR or VRPorn to the main VRHush site')" :delay="500" multilined >
                  <div class="control is-expanded">
                    <div class="select is-fullwidth">
                      <select v-model="listOfMainSites">
                        <option></option>
                        <option v-for="t in listOfMainSites" :key="t">{{ t }}</option>
                      </select>
                    </div>
                  </div>
              </b-tooltip>
            </b-field>
            <b-tooltip :label="$t('Restart XBVR to load new Sites')" :delay="500" type="is-warning">
              <b-field>
                <b-button type="is-primary" :disabled="!scraperFieldsValid" @click="saveScraper">Save</b-button>
              </b-field>
            </b-tooltip>
          </section>
        </div>
      </div>

      <div class="columns" v-if="activeTab == 3">
        <div class="column">
          <section>
            <b-field>
              <b-tooltip :label="$t('Scenes from Alternate Sites will be matched after Scene Scraping')" :delay="500">
                <b-switch v-model="linkScenesAfterSceneScraping" type="is-default">
                  Link Scenes after Scene Scraping
                </b-switch>
              </b-tooltip>
            </b-field>
            <b-field>
              <b-tooltip :label="$t('If a file is not matched to a scene, then try scenes from Alternate Sites')" :delay="500">
                <b-switch v-model="useAltSrcInFileMatching" type="is-default">
                  Include Scenes from Alternate Sites in File Matching
                </b-switch>
              </b-tooltip>
            </b-field>
            <b-field>
              <b-tooltip :label="$t('When filtering for Scenes with Scripts or sorting by Script Published Date, scenes from Alternate Sites will be included. Note: Slows these queries')" multiline :delay="500" type="is-warning">
                <b-switch v-model="useAltSrcInScriptFilters" type="is-default">
                  Include Scenes from Alternate Sites when filtering/sorting Scenes for Scripts
                </b-switch>
              </b-tooltip>
            </b-field>
            <b-tooltip :label="$t('Do not link scenes prior to the specified date.  The quality of metadata of older scenes is often poor and causes mismatches')" 
                :delay="500" type="is-primary" multilined size="is-large" position="is-bottom">
                <b-field label="Ignore Scenes Released Prior To">
                  <b-datepicker v-model="ignoreReleasedBefore" :icon-right="ignoreReleasedBefore ? 'close-circle' : ''" icon-right-clickable @icon-right-click="ignoreReleasedBefore = null">
                    <b-button
                        label="Today"
                        type="is-primary"
                        icon-left="calendar-today"
                        @click="ignoreReleasedBefore = new Date()" />

                    <b-button
                        label="Clear"
                        type="is-danger"
                        icon-left="close"
                        outlined
                        @click="ignoreReleasedBefore = null" />
                  </b-datepicker>
                </b-field>
              </b-tooltip>
            <b-field>              
              <b-button type="is-primary" @click="clearAltSrcKeepEdits" style="margin-right: 1em;">Clear scene links - keep edits</b-button>
              <b-button type="is-primary" @click="clearAltSrc" style="margin-right: 1em;">Clear scene links</b-button>
              <b-button type="is-primary" @click="relinkAltSrc" style="margin-right: 1em;">Re-link scenes</b-button>
            </b-field>
            <b-field>
              <b-button type="is-primary" @click="save">Save</b-button>
            </b-field>
          </section>
        </div>
      </div>

    </div>
  </div>
</template>

<script>
import ky from 'ky'
export default {
  name: 'InterfaceAdvanced',
  mounted () {    
    this.$store.dispatch('optionsAdvanced/load')    
  },
  data () {
    return {
      activeTab: 0,
      scraperUrl: '',
      scraperName: '',
      scraperCompany: '',
      scraperAvatar: '',
      scraperFieldsValid: false,
      masterSiteId: '',
      }
  },
  methods: {
    save () {      
      this.$store.dispatch('optionsAdvanced/save')
    },
    validateScraperFields() {      
      this.scraperFieldsValid=false
      if (this.scraperName != "") {
        if (this.scraperUrl.startsWith("https://") || this.scraperUrl.startsWith("http://") ) {
          if (this.scraperUrl.includes("povr.com") || this.scraperUrl.includes("sexlikereal.com") || this.scraperUrl.includes("vrphub.com") || this.scraperUrl.includes("vrporn.com")) {
            this.scraperFieldsValid=true
          }
        }
      }
    },
    saveScraper () {
       ky.put('/api/options/custom-sites/create', {
        json: {
          scraperUrl: this.scraperUrl,
          scraperName: this.scraperName,
          scraperCompany: this.scraperCompany,
          scraperAvatar: this.scraperAvatar,
          masterSiteId: this.masterSiteId,
        }
      })

    },
   stashdb () {
      ky.get('/api/extref/stashdb/run_all')
    },
    scrapeXbvrActors() {
      ky.get('/api/extref/generic/scrape_all')
    },
    clearAltSrcKeepEdits () {
      ky.delete(`/api/extref/delete_extref_source_links/keep_manual`, { json: {external_source: 'alternate scene %'} });
    },
    clearAltSrc () {
      ky.delete(`/api/extref/delete_extref_source_links/all`, { json: {external_source: 'alternate scene %'} });
    },
    relinkAltSrc () {
      ky.get('/api/task/relink_alt_aource_scenes')
    },
  },
  computed: {
    showInternalSceneId: {
      get () {
        return this.$store.state.optionsAdvanced.advanced.showInternalSceneId
      },
      set (value) {
        this.$store.state.optionsAdvanced.advanced.showInternalSceneId = value
      }
    },
    showHSPApiLink: {
      get () {        
        return this.$store.state.optionsAdvanced.advanced.showHSPApiLink
      },
      set (value) {
        this.$store.state.optionsAdvanced.advanced.showHSPApiLink = value
      },
    },
    showSceneSearchField: {
      get () {
        return this.$store.state.optionsAdvanced.advanced.showSceneSearchField
      },
      set (value) {
        this.$store.state.optionsAdvanced.advanced.showSceneSearchField = value
      },
    },
    stashApiKey: {
      get () {
        return this.$store.state.optionsAdvanced.advanced.stashApiKey
      },
      set (value) {
        this.$store.state.optionsAdvanced.advanced.stashApiKey = value
      }
    },
    dmmApiId: {
      get () {
        return this.$store.state.optionsAdvanced.advanced.dmmApiId
      },
      set (value) {
        this.$store.state.optionsAdvanced.advanced.dmmApiId = value
      }
    },
    dmmAffiliateId: {
      get () {
        return this.$store.state.optionsAdvanced.advanced.dmmAffiliateId
      },
      set (value) {
        this.$store.state.optionsAdvanced.advanced.dmmAffiliateId = value
      }
    },
    useImperialEntry: {
      get () {
        return this.$store.state.optionsAdvanced.advanced.useImperialEntry
      },
      set (value) {
        this.$store.state.optionsAdvanced.advanced.useImperialEntry = value

      }
    },
    scrapeActorAfterScene: {
      get () {
        return this.$store.state.optionsAdvanced.advanced.scrapeActorAfterScene
      },
      set (value) {
        this.$store.state.optionsAdvanced.advanced.scrapeActorAfterScene = value

      }
    },
    useAltSrcInFileMatching: {
      get () {
        return this.$store.state.optionsAdvanced.advanced.useAltSrcInFileMatching
      },
      set (value) {
        this.$store.state.optionsAdvanced.advanced.useAltSrcInFileMatching = value
      }
    },
    linkScenesAfterSceneScraping: {
      get () {
        return this.$store.state.optionsAdvanced.advanced.linkScenesAfterSceneScraping
      },
      set (value) {
        this.$store.state.optionsAdvanced.advanced.linkScenesAfterSceneScraping = value
      }

    },
    useAltSrcInScriptFilters: {
      get () {
        return this.$store.state.optionsAdvanced.advanced.useAltSrcInScriptFilters
      },
      set (value) {
        this.$store.state.optionsAdvanced.advanced.useAltSrcInScriptFilters = value
      }
    },
    ignoreReleasedBefore: {
      get () {
        return new Date(this.$store.state.optionsAdvanced.advanced.ignoreReleasedBefore)
      },
      set (value) {
        this.$store.state.optionsAdvanced.advanced.ignoreReleasedBefore = value
      }
    },
    listOfMainSites: {
      get () {
        const items = this.$store.state.optionsSites.items
        let sites = []
        for (let i=0; i < items.length; i++) {
          if (items[i].master_site_id == '') {
            sites.push(items[i].name)
          }      
        }
        return sites
      },
      set (value) {
        if (value == "") {
          this.masterSiteId=""
        } else {          
          const siteFound  = this.$store.state.optionsSites.items.find(site => site.master_site_id == '' && site.name==value);
          this.masterSiteId=siteFound.id
        }
      }
    },
    showMatchParamsOverlay () {
      return this.$store.state.overlay.sceneMatchParams.show
    },
    isLoading: function () {
      return this.$store.state.optionsAdvanced.loading
    }
  }
}
</script>

<style scoped>

</style>
