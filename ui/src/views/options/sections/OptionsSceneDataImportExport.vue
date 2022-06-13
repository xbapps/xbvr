<template>
  <div>
    <div class="content">
      <h3>{{$t('Export scene data')}}</h3>
      <p>
        {{$t('Scrap scene data, and  export it.')}}
      </p>
      <b-button type="is-primary" @click="exportContent">{{$t('Export content bundle')}}</b-button>
    </div>
    <hr/>
    <div class="content">
      <h3>{{$t("Backup/Restore database data")}}</h3> 
      <b-field label="Bundle version">
        <select v-model="formatVersion">
          <option>1</option>
          <option>2</option>          
        </select>        
      </b-field>
      <p v-show="formatVersion === '2'" >Include</p>
      <b-field v-show="formatVersion === '2'" >
        <b-switch v-model="allSites" ><p>{{ allSites ? 'All Sites' : 'Only Selected Sites' }}</p></b-switch>
      </b-field>        
      <div class="block" v-show="formatVersion === '2'" >
        <b-field grouped>
          <b-field label="Scenes">
            <b-switch v-model="includeScenes"><p>{{ includeScenes ? 'Included' : 'Excluded' }}</p></b-switch>
          </b-field>
          <b-field label="File Links">
            <b-switch v-model="includeFileLinks"><p>{{ includeFileLinks ? 'Included' : 'Excluded' }}</p></b-switch>
          </b-field>
          <b-field label="Cuepoints">
            <b-switch v-model="includeCuepoints"><p>{{ includeCuepoints ? 'Included' : 'Excluded' }}</p></b-switch>
          </b-field>
          <b-field label="Watch History">
            <b-switch v-model="includeHistory"><p>{{ includeHistory ? 'Included' : 'Excluded' }}</p></b-switch>
          </b-field>
          <b-field label="Edits">
            <b-switch v-model="includeActions"><p>{{ includeActions ? 'Included' : 'Excluded' }}</p></b-switch>
          </b-field>
        </b-field>
      </div>
      <div class="block" v-show="formatVersion === '2'" >
        <b-field grouped>
          <b-field label="Saved Searches">
            <b-switch v-model="includePlaylists"><p>{{ includePlaylists ? 'Included' : 'Excluded' }}</p></b-switch>
          </b-field>
          <b-field label="Storage Paths">
            <b-switch v-model="includeVolumes"><p>{{ includeVolumes ? 'Included' : 'Excluded' }}</p></b-switch>
          </b-field>
        </b-field>
      </div>
      <b-field grouped>
        <b-button type="is-primary" @click="backupContent">{{$t('Backup content bundle')}}</b-button>
      </b-field>
      <b-field grouped>
        <div class="button is-button is-primary" v-on:click="restoreContent">{{$t('Restore content bundle')}}</div>
          <b-input v-model="backupBundleURL" :placeholder="$t('Restore Bundle URL')" type="search" icon="web"></b-input>
          <b-field v-show="formatVersion === '2'" >
            <b-switch v-model="overwrite"><p>{{ overwrite ? 'New+Overwrite' : 'New Only' }}</p></b-switch>
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
      includeScenes: 'true',
      includeHistory: 'true',
      includeFileLinks: 'true',
      includeCuepoints: 'true',
      includeActions: 'true',
      includePlaylists: 'true',
      includeVolumes: 'true',
      overwrite: 'true',
      allSites: 'true',
      formatVersion: '2',
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
        ky.get('/api/task/bundle/restore', { searchParams: {formatVersion:this.formatVersion, allSites: this.allSites, url: this.backupBundleURL, inclScenes: this.includeScenes, inclHistory: this.includeHistory, inclLinks: this.includeFileLinks, inclCuepoints: this.includeCuepoints, inclActions: this.includeActions, inclPlaylists: this.includePlaylists, inclVolumes: this.includeVolumes, overwrite: this.overwrite } })
      }
    },
    backupContent () {
      ky.get('/api/task/bundle/backup', { searchParams: {formatVersion: this.formatVersion, allSites: this.allSites, inclScenes: this.includeScenes, inclHistory: this.includeHistory, inclLinks: this.includeFileLinks, inclCuepoints: this.includeCuepoints, inclActions: this.includeActions, inclPlaylists: this.includePlaylists, inclVolumes: this.includeVolumes } })
    }
  }         
}

</script>
