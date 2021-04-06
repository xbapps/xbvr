<template>
  <div>
    <div class="content">
      
      <h3>{{$t('Export funscripts')}}</h3>
      <p>
        {{$t('Here you can download a ZIP file containing a funscript for each scripted scene. The files are named by scene title and scene id, just like they appear in the DeoVR interface. If a scene has multiple scripts you can choose a preferred script in the scene details view. Otherwise, the last added script is chosen.')}}
      </p>
      <p>
        {{$t('Note that the filenames are not compatible with DLNA.')}}
      </p>
      <p>
        {{$t('To use this export with DeoVR: Unzip and put the files in the Interactive folder on your device.')}}
      </p>
      <p>
        {{$t('To use this export with ScriptPlayer: Unzip and put the files in a folder of your choice. In the ScriptPlayer settings, add this folder in the Paths section, then connect to DeoVR.')}}
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
