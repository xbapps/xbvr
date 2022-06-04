<template>
  <div>
    <div class="content">
      <h3>{{$t("Import scene data")}}</h3>
      <p>
        {{$t("You can import existing content bundles in JSON format from URL.")}}
      </p>
      <b-field grouped>
        <b-input v-model="bundleURL" :placeholder="$t('Bundle URL')" type="search" icon="web"></b-input>
        <div class="button is-button is-primary" v-on:click="importContent">{{$t('Import content bundle')}}</div>
      </b-field>
      <hr/>
    </div>
    <div class="content">
      <h3>{{$t('Export scene data')}}</h3>
      <p>
        {{$t('If you already have scraped scene data, you can export it below.')}}
      </p>
      <b-button type="is-primary" @click="exportContent">{{$t('Export content bundle')}}</b-button>
    </div>
    <hr/>
    <div class="content">
      <h3>{{$t("Backup/Restore  scene data")}}</h3>
      <p>
        {{$t("You can restore data from a backup bundle in JSON format from URL.")}}
      </p>      
      <p>Include</p>
      <b-field>
        <b-switch v-model="allSites">
          <p v-if="allSites">All Sites</p>
          <p v-else>Only Selected Sites</p>
        </b-switch>
      </b-field>        
      <div class="block">
        <b-field grouped>
          <b-field label="Scenes">
            <b-switch v-model="restoreScenes">
              <p v-if="restoreScenes">Included</p>
              <p v-else>Excluded</p>
            </b-switch>
          </b-field>
          <b-field label="File Links">
            <b-switch v-model="restoreFileLinks">
              <p v-if="restoreFileLinks">Included</p>
              <p v-else>Excluded</p>
            </b-switch>
          </b-field>
          <b-field label="Cuepoints">
            <b-switch v-model="restoreCuepoints">
              <p v-if="restoreCuepoints">Included</p>
              <p v-else>Excluded</p>
            </b-switch>
          </b-field>
          <b-field label="History">
            <b-switch v-model="restoreHistory">
              <p v-if="restoreHistory">Included</p>
              <p v-else>Excluded</p>
            </b-switch>
          </b-field>
          <b-field label="Actions">
            <b-switch v-model="restoreActions">
              <p v-if="restoreActions">Included</p>
              <p v-else>Excluded</p>
            </b-switch>
          </b-field>
        </b-field>
      </div>
      <div class="block">
        <b-field grouped>
          <b-field label="Playlists">
            <b-switch v-model="restorePlaylists">
              <p v-if="restorePlaylists">Included</p>
              <p v-else>Excluded</p>
            </b-switch>
          </b-field>
          <b-field label="Media Paths">
            <b-switch v-model="restoreVolumes">
              <p v-if="restoreVolumes">Included</p>
              <p v-else>Excluded</p>
            </b-switch>
          </b-field>
        </b-field>
      </div>
      <b-field grouped>
        <b-button type="is-primary" @click="backupContent">{{$t('Backup content bundle')}}</b-button>
      </b-field>
      <b-field grouped>
        <div class="button is-button is-primary" v-on:click="restoreContent">{{$t('Restore content bundle')}}</div>
          <b-input v-model="backupBundleURL" :placeholder="$t('Restore Bundle URL')" type="search" icon="web"></b-input>
          <b-field>
            <b-switch v-model="overwrite">
              <p v-if="overwrite">New+Overwrite</p>
              <p v-else>New Only</p>
            </b-switch>
          </b-field>        
      </b-field>
    </div>
  </div>
</template>

<script>
import { isThisMinute } from 'date-fns'
import ky from 'ky'
export default {
  name: 'OptionsSceneDataImportExport',
  data () {
    return {
      bundleURL: '',
      backupBundleURL: '',
      restoreScenes: 'true',
      restoreHistory: 'true',
      restoreFileLinks: 'true',
      restoreCuepoints: 'true',
      restoreActions: 'true',
      restorePlaylists: 'true',
      restoreVolumes: 'true',
      overwrite: 'true',
      allSites: 'true',
    }
  },
  methods: {
    importContent () {
      if (this.bundleURL !== '') {
        ky.get('/api/task/bundle/import', { searchParams: { url: this.bundleURL } })
      }
    },
    exportContent () {
      ky.get('/api/task/bundle/export')
    },
    restoreContent () {
      if (this.backupBundleURL !== '') {
        ky.get('/api/task/bundle/restore', { searchParams: {allSites: this.allSites, url: this.backupBundleURL, inclScenes: this.restoreScenes, inclHistory: this.restoreHistory, inclLinks: this.restoreFileLinks, inclCuepoints: this.restoreCuepoints, inclActions: this.restoreActions, inclPlaylists: this.restorePlaylists, inclVolumes: this.restoreVolumes, overwrite: this.overwrite } })
      }
    },
    backupContent () {
      ky.get('/api/task/bundle/backup', { searchParams: {allSites: this.allSites, inclScenes: this.restoreScenes, inclHistory: this.restoreHistory, inclLinks: this.restoreFileLinks, inclCuepoints: this.restoreCuepoints, inclActions: this.restoreActions, inclPlaylists: this.restorePlaylists, inclVolumes: this.restoreVolumes } })
    }
  }
}
</script>
