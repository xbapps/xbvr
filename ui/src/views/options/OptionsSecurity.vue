<template>
  <div>
    <div class="content">
      <h3>{{$t("Enable DLNA")}}</h3>
      <p>
        {{$t("You can enable/disable DLNA access")}}
      </p>
      <div class="switch">
        <b-switch :value="dlnaSettings.dlnaEnabled" v-model="dlnaEnabled" @input="$store.dispatch('optionsSecurity/enableDLNA', dlnaEnabled)"/>
      </div>
      <hr/>
    </div>
  </div>
</template>

<script>
  import ky from "ky";

  export default {
    name: "OptionsSecurity",
    data() {
      return {
        dlnaEnabled: true
      };
    },
    computed: {
      dlnaSettings() {
        return this.$store.state.optionsSecurity.dlnaOptions;
      },
    },
  }
</script>
