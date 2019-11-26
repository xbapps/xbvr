<template>
  <div>
    <div class="content">
      <h3>{{$t("Enable DLNA")}}</h3>
      <p>
        {{$t("You can enable/disable DLNA access")}}
      </p>
      <div class="switch">
        <b-switch :value="dlnaOptions.dlna_enabled" @input="$store.dispatch('optionsSecurity/enableDLNA', !dlnaOptions.dlna_enabled)"/>
      </div>
      <hr/>
    </div>
  </div>
</template>

<script>
  import ky from "ky";

  export default {
    name: "OptionsSecurity",
    mounted() {
      this.$store.dispatch("optionsSecurity/load");
    },
    computed: {
      dlnaOptions: function () {
        return this.$store.state.optionsSecurity.dlna_options;
      },
    },
  }
</script>
