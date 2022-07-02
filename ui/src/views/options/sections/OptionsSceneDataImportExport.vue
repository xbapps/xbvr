<template>
  <div>
    <div class="content">
      <h3>{{$t("Import/Export database data")}}</h3>
      <p></p>
      <hr />
      <h4>System Options</h4>
      <div class="block">
        <b-field grouped>
          <b-tooltip
            label="Includes your Saved Search definitions"
            size="is-large" type="is-primary is-light" multilined :delay="1500" >
            <b-field label="Saved Searches">
              <b-switch v-model="includePlaylists"><p>{{ includePlaylists ? 'Included' : 'Excluded' }}</p></b-switch>
            </b-field>
          </b-tooltip>
          <b-tooltip
            label="Include Storage Path data setup in Options/Storage"
            size="is-large" type="is-primary is-light" multilined :delay="1500" >
            <b-field label="Storage Paths">
              <b-switch v-model="includeVolumes"><p>{{ includeVolumes ? 'Included' : 'Excluded' }}</p></b-switch>
            </b-field>
          </b-tooltip>
          <b-tooltip
            label="Include Site Enabled settings from Option/Scrappers"
            size="is-large" type="is-primary is-light" multilined :delay="1500" >
            <b-field label="Site Settings">
              <b-switch v-model="includeSites"><p>{{ includeSites ? 'Included' : 'Excluded' }}</p></b-switch>
            </b-field>
          </b-tooltip>
        </b-field>
      </div>
      <hr />
      <h4>Scene Options</h4>
      <b-field grouped>
          <b-tooltip
            label="Selected Sites includes scenes for sites enabled in Options/Scrapers. All sites does not filter based on sites."
            size="is-large" type="is-primary is-light" multilined :delay="1500" >
            <b-switch v-model="allSites" ><p>{{ allSites ? 'All Sites' : 'Selected Sites' }}</p></b-switch>
          </b-tooltip>
          <b-tooltip
            label="Only includes scenes matching the Saved Search criteria. Applys to the Export function, not the Import"
            size="is-large" type="is-primary is-light" multilined :delay="1500">
            <b-field>
              <span style="margin-left:2em"><p>Filter:</p></span>
              <b-select size="is-small"  expanded v-model="currentPlaylist" title="Addtional Scene Filtering" style="margin-left:1em">
                  <option v-for="(obj, idx) in this.$store.state.sceneList.playlists" :value="obj.id" :key="idx">
                    {{ obj.name }}
                  </option>
              </b-select>
            </b-field>
          </b-tooltip>
      </b-field>
      <div class="block">
        <b-field grouped>
          <b-tooltip
            label="Include the main scene data, eg title, site, cast, tags, filenames, images, favorites, star ratings, etc"
            size="is-large" type="is-primary is-light" multilined :delay="1500" >
            <b-field label="Scene Data">
              <b-switch v-model="includeScenes"><p>{{ includeScenes ? 'Included' : 'Excluded' }}</p></b-switch>
            </b-field>
          </b-tooltip>
          <b-tooltip
            label="Include Cuepoint data you have entered for a scene"
            size="is-large" type="is-primary is-light" multilined :delay="1500" >
            <b-field label="Cuepoints">
              <b-switch v-model="includeCuepoints"><p>{{ includeCuepoints ? 'Included' : 'Excluded' }}</p></b-switch>
            </b-field>
          </b-tooltip>
          <b-tooltip
            label="Include your history of watching scenes"
            size="is-large" type="is-primary is-light" multilined :delay="1500" >
            <b-field label="Watch History">
              <b-switch v-model="includeHistory"><p>{{ includeHistory ? 'Included' : 'Excluded' }}</p></b-switch>
            </b-field>
          </b-tooltip>
          <b-tooltip
            label="Include scene edit data. Edits allows XBVR to reapply your changes to scene data. These would be lost if a scene is rescraped"
            size="is-large" type="is-primary is-light" multilined :delay="1500" >
            <b-field label="Edits">
              <b-switch v-model="includeActions"><p>{{ includeActions ? 'Included' : 'Excluded' }}</p></b-switch>
            </b-field>
          </b-tooltip>
          <b-tooltip
            label="Include details of files matched to a scene."
            size="is-large" type="is-primary is-light" multilined :delay="1500" >
            <b-field label="Matched Files">
              <b-switch v-model="includeFileLinks"><p>{{ includeFileLinks ? 'Included' : 'Excluded' }}</p></b-switch>
            </b-field>
          </b-tooltip>
        </b-field>
      </div>
      <hr />
      <b-field grouped>
        <b-tooltip
            label="Select a file to upload and import."
            size="is-large" type="is-primary is-light" multilined :delay="1500">
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
        <b-tooltip
          label="New/Overwrite will overwite existing data as well as add new records, New will only add new records, existing data is not changed"
          size="is-large" type="is-primary is-light" multilined :delay="1500">
          <b-field style="margin-left:1em">
            <b-switch v-model="overwrite"><p>{{ overwrite ? 'New/Overwrite' : 'New Only' }}</p></b-switch>
          </b-field>
        </b-tooltip>
      </b-field>
      <b-field grouped>
          <b-tooltip
            label="Generating the data for a large number of scenes is time consuming, montior progress in the status messages in the top right of the browser."
            size="is-large" type="is-primary is-light" multilined :delay="1500">
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
      uploadData: ''
    }
  },
  computed: {
    route () {
      return this.$route
    }
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
          json: { allSites: this.allSites, inclScenes: this.includeScenes, inclHistory: this.includeHistory, inclLinks: this.includeFileLinks, inclCuepoints: this.includeCuepoints, inclActions: this.includeActions, inclPlaylists: this.includePlaylists, inclVolumes: this.includeVolumes, inclSites: this.includeSites, overwrite: this.overwrite, uploadData: this.uploadData }
        })
        this.file = null
      }
    },
    backupContent () {
      ky.get('/api/task/bundle/backup', { timeout: false, searchParams: { allSites: this.allSites, inclScenes: this.includeScenes, inclHistory: this.includeHistory, inclLinks: this.includeFileLinks, inclCuepoints: this.includeCuepoints, inclActions: this.includeActions, inclPlaylists: this.includePlaylists, inclVolumes: this.includeVolumes, inclSites: this.includeSites, playlistId: this.currentPlaylist, download: true } }).json().then(data => {
        const link = document.createElement('a')
        link.href = this.myUrl
        link.click()
      })
    }
  }
}

</script>
