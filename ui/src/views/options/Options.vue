<template>
  <div class="container">
    <div class="columns">

      <div class="column is-one-fifth custom-menu">
        <b-menu :accordion="false">
          <b-menu-list :label="$t('Options')">
            <b-menu-item :label="$t('Storage')" :active="active==='storage'" @click="setActive('storage')"/>
            <b-menu-item :label="$t('Previews')" :active="active==='previews'" @click="setActive('previews')"/>
            <!--            <b-menu-item :label="$t('Scheduled tasks')" :active="active==='tasks'" @click="setActive ('tasks')"/>-->
            <b-menu-item :label="$t('Cache')" :active="active==='cache'" @click="setActive('cache')"></b-menu-item>
          </b-menu-list>
          <b-menu-list :label="$t('Scene data')">
            <b-menu-item :label="$t('Scrapers')" :active="active==='data-scrapers'"
                         @click="setActive('data-scrapers')"/>
            <b-menu-item :label="$t('Data import/export')" :active="active==='data-import-export'"
                         @click="setActive('data-import-export')"/>
          </b-menu-list>
          <b-menu-list :label="$t('Interfaces')">
            <b-menu-item :label="$t('Web UI')" :active="active==='interface_web'" @click="setActive('interface_web')"/>
            <b-menu-item :label="$t('DLNA')" :active="active==='interface_dlna'" @click="setActive('interface_dlna')"/>
            <!--            <b-menu-item :label="$t('DeoVR')" :active="active==='interface_deovr'" @click="setActive('ui_deovr')"/>-->
          </b-menu-list>
        </b-menu>
      </div>

      <div class="column">
        <div style="padding-top:2em">
          <Storage v-show="active==='storage'"/>
          <Cache v-show="active==='cache'"/>
          <Previews v-show="active==='previews'"/>
          <SceneDataScrapers v-show="active==='data-scrapers'"/>
          <SceneDataImportExport v-show="active==='data-import-export'"/>
          <InterfaceWeb v-show="active==='interface_web'"/>
          <InterfaceDLNA v-show="active==='interface_dlna'"/>
        </div>
      </div>

    </div>
  </div>
</template>

<script>
import InterfaceWeb from './sections/InterfaceWeb'
import Storage from './sections/Storage'
import SceneDataScrapers from './sections/OptionsSceneDataScrapers'
import SceneDataImportExport from './sections/OptionsSceneDataImportExport'
import InterfaceDLNA from './sections/InterfaceDLNA.vue'
import Cache from './sections/Cache.vue'
import Previews from './sections/Previews.vue'

export default {
  components: { Storage, SceneDataScrapers, SceneDataImportExport, InterfaceWeb, InterfaceDLNA, Cache, Previews },
  data: function () {
    return {
      active: 'storage'
    }
  },
  methods: {
    setActive: function (e) {
      this.active = e
    }
  }
}
</script>
