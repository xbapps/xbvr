<template>
  <div>
    <div class="content">
      <h3>{{$t("Import/Export database data")}}</h3>
      <b-tabs v-model="activeTab" size="medium" type="is-boxed" style="margin-left: 0px" id="importexporttab">
            <b-tab-item label="Import" icon="upload"/>
            <b-tab-item label="Export" icon="download"/>
        </b-tabs>
      <b-tabs v-model="activeSubTab" size="medium" type="is-boxed" style="margin-left: 0px" id="importexporttab">
            <b-tab-item label="Scene Data"/>
            <b-tab-item label="Actor Data"/>
            <b-tab-item label="Settings/Misc Data"/>
        </b-tabs>
      <h4 v-if="activeSubTab==0">{{ isImport ? "Import Scene Data" : "Export Scene Data"}}</h4>
      <b-field grouped v-if="activeSubTab == 0">
          <b-tooltip
            label="Select which studios are considered"
            size="is-large" type="is-primary is-light" multilined :delay="1000" >
              <b-radio v-model="allSites"
                name="allSites"
                native-value="true">
                All Studios
            </b-radio>
            <b-radio v-model="allSites"
                name="allSites"
                native-value="false">
                Only Studios enabled on Scrapers page
            </b-radio>
          </b-tooltip>
      </b-field>
      <b-field v-if="activeSubTab == 0">
          <b-tooltip
            :label="$t('You should exclude Studios from your Custom list (scrapers.json) if sharing data with others')"
            size="is-large" type="is-danger is-light" multilined :delay="100" >
            <b-switch v-model="onlyIncludeOfficalSites">{{$t("Only include offical studios")}}</b-switch>
          </b-tooltip>
      </b-field>
      <b-field v-if="isExport && activeSubTab == 0">
          <b-tooltip
            label="Only includes scenes matching the Saved Search criteria."
            size="is-large" type="is-primary is-light" multilined :delay="1000">
            <b-field style="margin-top:5px">
              <span style="margin-right:1em"><p>Filter by Saved Search:</p></span>
              <b-select placeholder="Saved Search" size="is-small" expanded v-model="currentPlaylist">
                  <option v-for="(obj, idx) in this.$store.state.sceneList.playlists" :value="obj.id" :key="idx">
                    {{ obj.name }}
                  </option>
              </b-select>
            </b-field>
          </b-tooltip>
        </b-field>
      <div class="block" style="margin-top:20px" v-if="activeSubTab == 0">
        <b-field>
          <b-tooltip
            label="Include the main scene data, eg title, site, cast, tags, filenames, images, favorites, star ratings, etc"
            size="is-large" type="is-primary is-light" multilined :delay="1000" >
            <b-switch v-model="includeScenes">Include Scene Data</b-switch>
          </b-tooltip>
        </b-field>
        <b-field>
          <b-tooltip
            label="Include Cuepoint data you have entered for a scene"
            size="is-large" type="is-primary is-light" multilined :delay="1000" >
            <b-switch v-model="includeCuepoints">Include Cuepoints</b-switch>
          </b-tooltip>
        </b-field>
        <b-field>
          <b-tooltip
            label="Include your history of watched scenes"
            size="is-large" type="is-primary is-light" multilined :delay="1000" >
            <b-switch v-model="includeHistory">Include Watch History</b-switch>
          </b-tooltip>
        </b-field>
        <b-field>
          <b-tooltip
            label="Include scene edit data. Edits allows XBVR to reapply your changes to scene data. These would be lost if a scene is rescraped"
            size="is-large" type="is-primary is-light" multilined :delay="1000" >
            <b-switch v-model="includeActions">Include Scene Edits</b-switch>
          </b-tooltip>
        </b-field>
        <b-field>
          <b-tooltip
            label="Include details of files matched to a scene."
            size="is-large" type="is-primary is-light" multilined :delay="1000" >
            <b-switch v-model="includeFileLinks"><p>Include Matched Files</p></b-switch>
          </b-tooltip>
        </b-field>
        <b-button type="is-info is-small" style="margin-bottom: 1em;"  @click="toggleSceneIncludes()">Toggle Includes</b-button>
      </div>
      <hr />
      <div v-if="activeSubTab==1">
        <h4>{{ isImport ? "Import Actor Data" : "Export Actor Data"}}</h4>
        <b-field>
          <b-tooltip
            label="Includes Actors (note new actors are not created, New/Existing will apply to the fields on an existing actor record.)"
            size="is-large" type="is-primary is-light" multilined :delay="1000" >
            <b-switch v-model="includeActors">Include Actors</b-switch>
          </b-tooltip>
        </b-field>
        <b-field>
          <b-tooltip
            label="Includes your Actor Aka Groups"
            size="is-large" type="is-primary is-light" multilined :delay="1000" >
            <b-switch v-model="includeActorAkas">Include Actor Aka Groups</b-switch>
          </b-tooltip>
        </b-field>
        <b-field>
          <b-tooltip
            label="Includes your Actor Edits"
            size="is-large" type="is-primary is-light" multilined :delay="1000" >
            <b-switch v-model="inclActorActions">Include Actor Edits</b-switch>
          </b-tooltip>
        </b-field>
        <b-button type="is-info is-small" style="margin-bottom: 1em;"  @click="toggleActorIncludes()">Toggle Includes</b-button>
      </div>
      <h4 v-if="activeSubTab==2">{{ isImport ? "Import Settings" : "Export Settings"}}</h4>
      <div class="block" v-if="activeSubTab == 2">
        <b-field>
          <b-tooltip
            label="Includes your Tag Groups"
            size="is-large" type="is-primary is-light" multilined :delay="1000" >
            <b-switch v-model="includeTagGroups">Include Tag Groups</b-switch>
          </b-tooltip>
        </b-field>
        <b-field>
          <b-tooltip
            label="Includes your Saved Search definitions"
            size="is-large" type="is-primary is-light" multilined :delay="1000" >
            <b-switch v-model="includePlaylists">Include Saved Searches</b-switch>
          </b-tooltip>
        </b-field>
        <b-field>
          <b-tooltip
            label="Include Storage Path data setup in Options/Storage"
            size="is-large" type="is-primary is-light" multilined :delay="1000" >
            <b-switch v-model="includeVolumes">Include Storage Paths</b-switch>
          </b-tooltip>
        </b-field>
        <b-field>
          <b-tooltip
            label="Include Studio Enabled settings from Option/Scrappers"
            size="is-large" type="is-primary is-light" multilined :delay="1000" >
            <b-switch v-model="includeSites">Include Scraper Settings</b-switch>
          </b-tooltip>
        </b-field>
        <div class="columns">
          <div class="column">
            <b-field>
              <b-tooltip
                label="Includes your External References"
                size="is-large" type="is-primary is-light" multilined :delay="1000" >
                <b-switch v-model="includeExternalReferences">Include External References</b-switch>
              </b-tooltip>
            </b-field>
          </div>
          <div class="column">
            <b-field :label="$t('Subset')" label-position="on-border" :addons="true" class="field-extra">
              <div class="control is-expanded">
                <div class="select is-fullwidth">
                  <select v-model="extRefSubset">
                    <option value="">{{ $t("Include All External References") }}</option>
                    <option value="manual_matched">{{ $t("Manual Matched Alternate Source Scenes") }}</option>
                    <option value="deleted_match">{{ $t("Deleted Alternate Source Sceness") }}</option>
                  </select>
                </div>
              </div>
            </b-field>
          </div>
        </div>
        <b-field>
          <b-tooltip
            :label="isImport ? 'Requires restarting XBVR once complete. Include XBVR Configuration Settings. Preview setting, task schedules, etc.' : 'Includes passowrds/access tokens. Includes XBVR Configuration Settings. Preview settings, task schedules, etc.'"  
            size="is-large" :type="isImport ? 'is-warning is-light' : 'is-danger is-light'" multilined :delay="300" >
            <b-switch v-model="includeConfig">Include Config Settings</b-switch>
          </b-tooltip>
        </b-field>
        <b-button type="is-info is-small" style="margin-bottom: 1em;"  @click="toggleSettingsIncludes()">Toggle Includes</b-button>
      </div>
      <hr />
      <b-field v-if="isImport">
        <b-tooltip
          label="Activate to overwite existing data, otherwise only new records will be added"
          size="is-large" type="is-primary is-light" multilined :delay="1000">
          <b-switch v-model="overwrite"><p>Overwrite existing data</p></b-switch>
        </b-tooltip>
      </b-field>
      <b-field>
        <b-tooltip v-if="isImport"
            label="Select a file to import."
            size="is-large" type="is-primary is-light" multilined :delay="1000">
          <b-field class="file is-primary" :class="{'has-name': !!file}">
            <b-upload v-model="file" class="file-label" icon-left="upload">
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
          <b-tooltip  v-if="activeTab == 1"
            label="Generating the data for a large number of scenes is time consuming, montior progress in the status messages in the top right of the browser."
            size="is-large" type="is-primary is-light" multilined :delay="1000">
            <b-button type="is-primary"  @click="backupContent" icon-left="download">Export
            </b-button>
          </b-tooltip>
        <b-tooltip style="margin-left: 10px"            
            :label="$t('Occasionaly test uploading your export bundles. Browser memory constraints may cause problems restoring large exports. Use this function to test if your browser can load an export.')"
            size="is-large" type="is-primary is-light" multilined :delay="1000">
          <b-field class="file is-primary" :class="{'has-name': !!file}">
            <b-upload v-model="testfile" class="file-label">
                <span class="file-cta">
                    <b-icon class="file-icon" icon="upload" size="is-small"></b-icon>
                    <span class="file-label">Test</span>
                </span>
                <span class="file-name" v-if="progressMsg">
                    {{ progressMsg }}
                </span>
            </b-upload>
          </b-field>
        </b-tooltip>
      </b-field>
    </div>
  </div>
</template>

<script>
import ky from 'ky'
export default {
  name: 'OptionsSceneDataImportExport',
  mounted () {
    this.$store.dispatch('sceneList/filters')
  },
  data () {
    return {
      includeScenes: true,
      includeHistory: true,
      includeFileLinks: true,
      includeCuepoints: true,
      includeActions: true,
      includePlaylists: true,
      includeVolumes: true,
      includeSites: true,
      includeConfig: false,
      includeActorAkas: true,
      includeExternalReferences: true,
      includeTagGroups: true,
      includeActors: true,
      inclActorActions: true,
      overwrite: true,
      allSites: "true",
      onlyIncludeOfficalSites: false,
      currentPlaylist: '0',
      extRefSubset: '',
      myUrl: '/download/xbvr-content-bundle.json',
      file: null,
      testfile: null,
      progressMsg:"",
      uploadData: '',
      activeTab: 0,
      activeSubTab: 0
    }
  },
  computed: {
    route () {
      return this.$route
    },
    isImport() {
      return this.activeTab == 0
    },
    isExport() {
      return this.activeTab == 1
    },
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
              this.restoreContent()
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
    testfile: function (o, n) {
      try {        
        this.$buefy.toast.open({message: `Loading: ` + this.testfile.name, type: 'is-primary', duration: 30000})
        if (this.testfile != null) {
          this.progressMsg = "Uploading " + this.testfile.name
          const reader = new FileReader()
          reader.onload = (event) => {
            try {
              this.progressMsg = "File uploaded, converting to Json " + this.testfile.name
              this.uploadData = JSON.stringify(JSON.parse(event.target.result))          
              this.progressMsg = ""
              this.$buefy.toast.open({message: `Success Loading: ` + this.testfile.name, type: 'is-success', duration: 5000})
          } catch (error) {
            this.progressMsg = "Error: " + error.message            
            this.$buefy.toast.open({message: `Error:  ${error.message}`, type: 'is-danger', duration: 30000})    
          }

          }
          reader.readAsText(this.testfile)
        }      
      } catch (error) {        
        this.progressMsg = "Error: " + error.message
        this.$buefy.toast.open({message: `Error:  ${error.message}`, type: 'is-danger', duration: 30000})    
      }
    }
  },
  methods: {
    restoreContent () {
      if (this.uploadData !== '') {
        // put up a starting msg, as large files can cause it to appear to hang
        this.$store.state.messages.lastScrapeMessage = 'Starting restore'
        ky.post('/api/task/bundle/restore', {
          json: { allSites: this.allSites == "true", onlyIncludeOfficalSites: this.onlyIncludeOfficalSites, inclScenes: this.includeScenes, inclHistory: this.includeHistory, 
          inclLinks: this.includeFileLinks, inclCuepoints: this.includeCuepoints, inclActions: this.includeActions, inclPlaylists: this.includePlaylists, inclActorAkas: this.includeActorAkas, inclTagGroups: this.includeTagGroups, 
          inclVolumes: this.includeVolumes, inclExtRefs: this.includeExternalReferences, inclSites: this.includeSites, inclActors: this.includeActors,inclActorActions: this.inclActorActions, 
          inclConfig: this.includeConfig, extRefSubset: this.extRefSubset, overwrite: this.overwrite, uploadData: this.uploadData }
        })
        this.file = null
      }
    },
    backupContent () {      
      ky.get('/api/task/bundle/backup', { timeout: false, searchParams: { allSites: this.allSites == "true", onlyIncludeOfficalSites: this.onlyIncludeOfficalSites, inclScenes: this.includeScenes, inclHistory: this.includeHistory,
           inclLinks: this.includeFileLinks, inclCuepoints: this.includeCuepoints, inclActions: this.includeActions, inclPlaylists: this.includePlaylists, inclActorAkas: this.includeActorAkas, inclTagGroups: this.includeTagGroups, 
           inclVolumes: this.includeVolumes, inclExtRefs: this.includeExternalReferences, inclSites: this.includeSites, inclActors: this.includeActors,inclActorActions: this.inclActorActions,
           inclConfig: this.includeConfig, extRefSubset: this.extRefSubset, playlistId: this.currentPlaylist, download: true } }).json().then(data => {      
        const link = document.createElement('a')
        link.href = this.myUrl
        link.click()
      })
    },
    toggleSceneIncludes () {
      this.includeScenes = !this.includeScenes
      this.includeCuepoints  = !this.includeCuepoints
      this.includeHistory = !this.includeHistory
      this.includeActions=!this.includeActions
      this.includeFileLinks=!this.includeFileLinks
    },
    toggleActorIncludes () {
      this.includeActors = !this.includeActors
      this.includeActorAkas = !this.includeActorAkas
      this.inclActorActions = !this.inclActorActions
    },
    toggleSettingsIncludes () {
      this.includeTagGroups = !this.includeTagGroups
      this.includePlaylists = !this.includePlaylists
      this.includeVolumes=!this.includeVolumes
      this.includeSites=!this.includeSites
      this.includeExternalReferences = !this.includeExternalReferences
      this.includeConfig=!this.includeConfig
    },
  }
}

</script>

<style>
#importexporttab ul[role="tablist"] {
    margin-left: 0px;
}

#importexporttab section.tab-content {
    display:none;
}

</style>