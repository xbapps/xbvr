<template>
  <div>
    <div class="content">
      <h3>{{$t("Import/Export database data")}}</h3>
      <b-tabs v-model="activeTab" size="medium" type="is-boxed" style="margin-left: 0px" id="importexporttab">
            <b-tab-item label="Import" icon="upload"/>
            <b-tab-item label="Export" icon="download"/>
        </b-tabs>
      <h4>{{ isImport ? "Import Scene Data" : "Export Scene Data"}}</h4>
      <b-field grouped>
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
      <b-field v-if="isExport">
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
      <div class="block" style="margin-top:20px">
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
      </div>
      <hr />
      <h4>{{ isImport ? "Import Settings" : "Export Settings"}}</h4>
      <div class="block">
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
      </div>
      <hr />
      <b-field v-if="isImport">
        <b-tooltip
          label="Activate to overwite existing data, otherwise only new records will be added"
          size="is-large" type="is-primary is-light" multilined :delay="1000">
          <b-switch v-model="overwrite"><p>Overwrite existing data</p></b-switch>
        </b-tooltip>
      </b-field>
      <b-field v-if="isImport">
        <b-tooltip
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
      </b-field>
      <b-field v-if="activeTab == 1">
          <b-tooltip
            label="Generating the data for a large number of scenes is time consuming, montior progress in the status messages in the top right of the browser."
            size="is-large" type="is-primary is-light" multilined :delay="1000">
            <b-button type="is-primary"  @click="backupContent" icon-left="download">Export
            </b-button>
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
      overwrite: true,
      allSites: true,
      currentPlaylist: '0',
      myUrl: '/download/xbvr-content-bundle.json',
      file: null,
      uploadData: '',
      activeTab: 0
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
      if (this.file != null) {
        const reader = new FileReader()
        reader.onload = (event) => {
          this.uploadData = JSON.stringify(JSON.parse(event.target.result))
          this.restoreContent()
        }
        reader.readAsText(this.file)
      }
    }
  },
  methods: {
    restoreContent () {
      if (this.uploadData !== '') {
        // put up a starting msg, as large files can cause it to appear to hang
        this.$store.state.messages.lastScrapeMessage = 'Starting restore'
        ky.post('/api/task/bundle/restore', {
          json: { allSites: this.allSites == "true", inclScenes: this.includeScenes, inclHistory: this.includeHistory, inclLinks: this.includeFileLinks, inclCuepoints: this.includeCuepoints, inclActions: this.includeActions, inclPlaylists: this.includePlaylists, inclVolumes: this.includeVolumes, inclSites: this.includeSites, overwrite: this.overwrite, uploadData: this.uploadData }
        })
        this.file = null
      }
    },
    backupContent () {
      ky.get('/api/task/bundle/backup', { timeout: false, searchParams: { allSites: this.allSites == "true", inclScenes: this.includeScenes, inclHistory: this.includeHistory, inclLinks: this.includeFileLinks, inclCuepoints: this.includeCuepoints, inclActions: this.includeActions, inclPlaylists: this.includePlaylists, inclVolumes: this.includeVolumes, inclSites: this.includeSites, playlistId: this.currentPlaylist, download: true } }).json().then(data => {
        const link = document.createElement('a')
        link.href = this.myUrl
        link.click()
      })
    }
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