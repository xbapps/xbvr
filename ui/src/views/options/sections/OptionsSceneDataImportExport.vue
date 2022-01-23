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
      <hr/>
    </div>
    <div class="content">
      <h3>{{$t("Add custom scene")}}</h3>
      <p>
        {{$t("You can add a custom scene with a specific title.")}}
      </p>
      <b-field grouped>
        <b-input v-model="sceneTitle" :placeholder="$t('Scene title')" type="search" icon="plus"></b-input>
        <div class="button is-button is-primary" v-on:click="addScene">{{$t('Add custom scene')}}</div>
      </b-field>
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
    },
	addScene() {
      if (this.sceneTitle !== '') {
        ky.post('/api/scene/create', { searchParams: { title: this.sceneTitle }, json: {} })
      }
	}
  }
}
</script>
