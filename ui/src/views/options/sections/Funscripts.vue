<template>
  <div>
    <div class="content">
      <h3>{{ $t("Export funscripts") }}</h3>
      <p>
        {{$t('Here you can download a ZIP file containing a funscript for each scripted scene. The file names include scene title and scene id, as expected by DeoVR. If a scene has multiple scripts you can choose a preferred script in the scene details view. Otherwise, the most recently added script is chosen.')}}
      </p>
      <p>
        {{ $t("Note that the filenames are not compatible with DLNA.") }}
      </p>
      <p>
        {{
          $t(
            "To use this export with DeoVR: Unzip and put the files in the Interactive folder on your device."
          )
        }}
      </p>
      <p>
        {{
          $t(
            "To use this export with ScriptPlayer: Unzip and put the files in a folder of your choice. In the ScriptPlayer settings, add this folder in the Paths section, then connect to DeoVR."
          )
        }}
      </p>
      <hr />
      <p><strong>Download funscripts for DeoVR</strong></p>
      <p>
        <b-button
          type="is-primary"
          @click="exportAllFunscripts"
          :disabled="countTotal === 0"
          icon-left="download"
          >{{ $t("Download all funscripts") }} ({{ countTotal }})</b-button
        >
      </p>
      <p>
        <b-button
          type="is-primary"
          @click="exportNewFunscripts"
          :disabled="countUpdated === 0"
          icon-left="download"
          >{{ $t("Download changes since last export") }} ({{
            countUpdated
          }})</b-button
        >
      </p>
      <hr />
      <b-field>
        <b-switch v-model="scrapeFunscripts" type="is-default">
          <strong>Scrape for Available Funscripts</strong>
        </b-switch>
      </b-field>
      <b-field>
        <b-button type="is-primary" @click="save">Save</b-button>
      </b-field>
    </div>
  </div>
</template>

<script>
import ky from "ky";

export default {
  name: "Funscripts",
  mounted() {
    this.$store.dispatch("optionsFunscripts/load");
  },
  methods: {
    exportAllFunscripts() {
      const link = document.createElement("a");
      link.href = "/api/task/funscript/export-all";
      link.click();
    },
    exportNewFunscripts() {
      const link = document.createElement("a");
      link.href = "/api/task/funscript/export-new";
      link.click();
    },
    save () {
      this.$store.dispatch('optionsFunscripts/save')
    },
  },
  computed: {
    countTotal: function () {
      return this.$store.state.optionsFunscripts.countTotal;
    },
    countUpdated: function () {
      return this.$store.state.optionsFunscripts.countUpdated;
    },
    scrapeFunscripts: {
      get () {
        return this.$store.state.optionsFunscripts.optionsFunscripts.scrapeFunscripts
      },
      set (value) {
        this.$store.state.optionsFunscripts.optionsFunscripts.scrapeFunscripts = value
      },
    },
  },
};
</script>
