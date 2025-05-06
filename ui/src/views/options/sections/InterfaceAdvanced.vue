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
            <b-tab-item :label="$t('Cookies/Headers')"/>
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

      <!-- Heaaders/Cookies tab -->
      <div class="columns" v-if="activeTab == 4">
        <div class="column">
          <section>            
            <b-field>
              <p>
                <b><a href="https://github.com/xbapps/xbvr/wiki/Setting-Request-Headers,-Cookies-and-Body" target="_blank" rel="noreferrer">Domain Config</a></b>
                <a href="https://github.com/xbapps/xbvr/wiki/Setting-Request-Headers,-Cookies-and-Body" target="_blank" rel="noreferrer" style="margin-left: 1em;">Wiki</a>
              </p>
              <b-field>
                <b-tooltip label="Select a file to import config for a scraper or trailers"
                  type="is-primary is-light" :delay="1000" style="margin-left: 1em;" >
                  <b-field class="file is-primary" :class="{'has-name': !!file}">
                    <b-upload v-model="file" class="file-label" icon-left="upload" size="is-small">
                        <span class="file-cta">
                            <b-icon class="file-icon" icon="upload" size="is-small"></b-icon>
                            <span class="file-label">Import</span>
                        </span>
                        <span class="file-name" v-if="file">
                            {{ file.name }}
                        </span>
                    </b-upload>
                  </b-field>
                </b-tooltip>
                <b-button v-if="showConfigField" type="is-primary" style="margin-left: 1em;" @click="saveCollectorConfig">Save</b-button>
                <b-button v-if="showConfigField" type="is-primary" style="margin-left: 1em;" @click="deleteCollectorConfig" icon-right="delete"></b-button>
              </b-field>
              <b-autocomplete v-model="kvName" ref="autocompleteconfig" :data="filteredCollectorConfigList" :open-on-focus="true" :clearable="true" 
                placeholder="e.g. domainname-scraper or domainname-trailers " 
                @select="option => showConfigDetails(option)" 
                @select-header="showAddCollectorConfig"
                :selectable-header="true" >
                <template #header>
                    <a><span> Add new... </span></a>
                </template>
              </b-autocomplete>
            </b-field>
            <b-field v-if="showConfigField">
              <p><b>Headers</b><b-button style="margin-left: 1em;" @click="addHeaderRow" size="is-small"><b-icon pack="mdi" icon="plus" size="is-small"></b-icon></b-button></p>              
            </b-field>
            <b-table v-if="showConfigField" :data="headers" >
              <b-table-column field="key" :label="$t('Key')" width="200" v-slot="props">
                <b-input v-model="props.row.key" ></b-input>                
              </b-table-column>
              <b-table-column field="value" :label="$t('Value')" v-slot="props">
                <b-input v-model="props.row.value" ></b-input>                
              </b-table-column>
              <b-table-column field="value" v-slot="props">
                <b-button @click="delHeaderRow(props)"><b-icon pack="fas" icon="trash" ></b-icon></b-button>  
              </b-table-column>
            </b-table>

            <b-field v-if="showConfigField" >
              <p><b>Cookies</b><b-button style="margin-left: 1em;"@click="addCookieRow" size="is-small"><b-icon pack="mdi" icon="plus" size="is-small"></b-icon></b-button></p>              
            </b-field>
            <b-table v-if="showConfigField" :data="cookies" >
              <b-table-column field="name" :label="$t('Key')" width="200" v-slot="props">
                <b-input v-model="props.row.name" ></b-input>                
              </b-table-column>
              <b-table-column field="value" :label="$t('Value')" v-slot="props">
                <b-input v-model="props.row.value" ></b-input>                
              </b-table-column>
              <b-table-column field="domain" :label="$t('Domain')" v-slot="props">
                <b-input v-model="props.row.domain" ></b-input>                
              </b-table-column>
              <b-table-column field="path" :label="$t('Path')" width="100" v-slot="props">
                <b-input v-model="props.row.path" ></b-input>                
              </b-table-column>
              <b-table-column field="host" :label="$t('Host')" v-slot="props">
                <b-input v-model="props.row.host" ></b-input>                
              </b-table-column>
              <b-table-column v-slot="props">
                <b-button @click="delCookieRow(props)"><b-icon pack="fas" icon="trash" ></b-icon></b-button>  
              </b-table-column>
            </b-table>

            <b-field  v-if="showConfigField" label="Request Body">
              <b-input v-model="body" type="textarea"></b-input>                
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
      kvName: "",
      headers: [],
      cookies: [],
      body: "",
      showCollectorConfigFields: false,
      file: null,
      uploadData: '',
    }
  },
  watch: {
    // when a file is selected, then this will fire the upload process
    file: function (o, n) {
      try {
        if (this.file != null) {
          const reader = new FileReader()
          reader.onload = (event) => {
            try {
              this.uploadData = JSON.stringify(JSON.parse(event.target.result))
              this.restoreCollectorConfig()
            } catch (error) {
              this.$buefy.toast.open({message: `Error:  ${error.message}`, type: 'is-danger', duration: 30000})    
            }
          }
          reader.readAsText(this.file)
        }
      } catch (error) {        
        this.$buefy.toast.open({message: `Error:  ${error.message}`, type: 'is-danger', duration: 30000})    
      }
    },
  },
  methods: {
    save () {      
      this.$store.dispatch('optionsAdvanced/save')
    },
    validateScraperFields() {      
      this.scraperFieldsValid=false
      if (this.scraperName != "") {
        if (this.scraperUrl.startsWith("https://") || this.scraperUrl.startsWith("http://") ) {
          if (this.scraperUrl.includes("povr.com") || this.scraperUrl.includes("sexlikereal.com") || this.scraperUrl.includes("vrphub.com") || this.scraperUrl.includes("vrporn.com") || this.scraperUrl.includes("stashdb.org") || this.scraperUrl.includes("realvr.com")) {
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
    showConfigDetails(option) {      
      let matched = this.$store.state.optionsAdvanced.advanced.collectorConfigs.find((config) => {
        return config.domain_key
          .toString()
          .toLowerCase()
          .trim()
          .indexOf(option.toLowerCase()) >= 0
      })
      if (matched == null ) return
      if (matched.config.headers==null) {
        this.headers = []
      } else {
        this.headers = matched.config.headers
      }
      if (matched.config.cookies==null) {
        this.cookies = []
      } else {
        this.cookies = matched.config.cookies
      }      
      this.body= matched.config.body
      this.showCollectorConfigFields = true
    },
    addHeaderRow(){
      this.headers.push({ key: "", value: "" });
    },
    delHeaderRow(props){      
      this.headers.splice(props.index,1)
    },
    addCookieRow(){
      this.cookies.push({ name: "", value: "", domain: "", path: "", host:"" });
    },
    delCookieRow(props){      
      this.cookies.splice(props.index,1)
    },
    saveCollectorConfig() {
      ky.post('/api/options/save-collector-config', {
        json: {
          domain_key: this.kvName,
          body: this.body,
          cookies: this.cookies,
          headers: this.headers,
          other: [],
        }
      })
      let row = this.$store.state.optionsAdvanced.advanced.collectorConfigs.find((config) => {
        return config.domain_key
          .toString()
          .toLowerCase()
          .trim()
          .indexOf(this.kvName.toString().toLowerCase()) >= 0
      })
      row.config.cookies = this.cookies
      row.config.headers=this.headers
      row.config.body = this.body      
    },
    showAddCollectorConfig() {
      this.$buefy.dialog.prompt({
                message: `Add new config`,
                inputAttrs: {
                    placeholder: 'domainname-scraper or domainname-trailers e.g. naughtyamerica-trailers',
                    maxlength: 20,
                    value: this.kvName
                },
                confirmText: 'Add',
                onConfirm: (value) => {                  
                    this.kvName=value
                    this.$store.state.optionsAdvanced.advanced.collectorConfigs.push({config: {body: "", cookies: [], headers: [],  other: ""},domain_key: value} )                    
                    this.$refs.autocompleteconfig.setSelected(value)
                    this.showCollectorConfigFields = true                    
                }
      })
    },
    async restoreCollectorConfig () {
      if (this.uploadData !== '') {
        try {
          const response = await ky.post('/api/options/save-collector-config', {
            json: JSON.parse(this.uploadData)
          })
        } catch (error) {
          this.$buefy.toast.open({message: `Error:  Failed to import file`, type: 'is-danger', duration: 30000})
          return
        }
        this.file = null
        this.$store.dispatch('optionsAdvanced/load')
      }
    },
    async deleteCollectorConfig() {
      const response = await ky.delete('/api/options/delete-collector-config', {
        json: {
          domain_key: this.kvName,
        }
      })
      this.kvName=""
      this.$store.dispatch('optionsAdvanced/load')
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
      },
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
    filteredCollectorConfigList () {
      // filter the list based on what has been entered so far
      if (this.$store.state.optionsAdvanced.advanced.collectorConfigs.length==0) return

      let matched = this.$store.state.optionsAdvanced.advanced.collectorConfigs.filter((config) => {
        return config.domain_key
          .toLowerCase()
          .trim()
          .indexOf(this.kvName.toLowerCase()) >= 0
      })
      if (matched.length == 1 && matched[0].domain_key.toLowerCase() == this.kvName.toLowerCase()){
        this.showCollectorConfigFields = true
        this.showConfigDetails(this.kvName)
      } else {
        this.showCollectorConfigFields = false
      }
      return matched.map(item => item.domain_key)
    },
    showConfigField () {
      return this.showCollectorConfigFields
    },
    isLoading: function () {
      return this.$store.state.optionsAdvanced.loading
    }
  }
}
</script>

<style scoped>

</style>
