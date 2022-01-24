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
  </div>
</template>

<script>
import ky from 'ky'
export default {
  name: 'OptionsSceneDataImportExport',
  data () {
    return {
      bundleURL: ''
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
    }
  }
}
</script>
