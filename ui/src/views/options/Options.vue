<template>
  <div class="container">
    <div class="columns">

      <div class="column is-one-fifth custom-menu">
        <b-menu>
          <b-menu-list :label="$t('Options')">
            <b-menu-item :label="$t('Storage')" :active="active==='folders'"
                         @click="setActive('folders')"/>
            <b-menu-item :label="$t('Scene data')" :expanded="true" @click="setActive('data-scrapers')">
              <b-menu-item :label="$t('Scrapers')" :active="active==='data-scrapers'" @click="setActive('data-scrapers')"/>
              <b-menu-item :label="$t('Data import/export')" :active="active==='data-import-export'"
                           @click="setActive('data-import-export')"/>
            </b-menu-item>
          </b-menu-list>
        </b-menu>
      </div>

      <div class="column">
        <div style="padding-top:2em">
          <Folders v-show="active==='folders'"/>
          <SceneDataScrapers v-show="active==='data-scrapers'"/>
          <SceneDataImportExport v-show="active==='data-import-export'"/>
          <SceneDataJAVR v-show="active==='data-javr'"/>
        </div>
      </div>

    </div>
  </div>
</template>

<script>
  import Folders from "./OptionsFolders.vue";
  import SceneDataScrapers from "./OptionsSceneDataScrapers.vue";
  import SceneDataImportExport from "./OptionsSceneDataImportExport";

  export default {
    components: {Folders, SceneDataScrapers, SceneDataImportExport},
    data: function () {
      return {
        active: "folders",
      }
    },
    methods: {
      setActive: function (e) {
        this.active = e;
      }
    },
  }
</script>
