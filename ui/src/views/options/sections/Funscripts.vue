<template>
  <div>
    <div class="content">
      
      <h3>{{$t('Export funscripts')}}</h3>
      <p>
        {{$t('Here you can download a zip file with a funscript for each scripted scene. The files are named with the scene title and the scene id. Scipted scenes in the DeoVR interface will be named accordingly. For scenes with multiple scripts you can choose a preferred one in the scene details view. Otherwise the last one added is chosen.')}}
      </p>
      <p>
        {{$t('This export won\'t work with DLNA.')}}
      </p>
      <p>
        {{$t('To use this export with DeoVR: Unzip and put the files in the Interactive folder on your device.')}}
      </p>
      <p>
        {{$t('To use this export with ScriptPlayer: Unzip and put the files in a folder of your choice. Add this folder in the Paths section in the settings. Connect to DeoVR.')}}
      </p>
      <b-button type="is-primary" @click="exportFunscripts" :disabled="count === 0">{{$t('Download funscripts for DeoVR')}}</b-button>
      <p>
        {{count}} scripted scenes available.
      </p>
    </div>
  </div>
</template>

<script>
import ky from 'ky'

export default {
  name: 'Funscripts',
  mounted () {
    this.$store.dispatch('optionsFunscripts/load')
  },
  methods: {
    exportFunscripts () {
      const link = document.createElement('a')
      link.href = '/api/task/funscript/export'
      link.click()
    },
  },
  computed: {
    count: function () {
      return this.$store.state.optionsFunscripts.count
    },
  }
}
</script>
