<template>
  <div class="container">
    <div class="tabs is-medium is-boxed">
      <ul>
        <li v-bind:class="{'is-active': active==='folders'}"><a v-on:click="setActive('folders')">Folders</a></li>
        <li v-bind:class="{'is-active': active==='sites'}"><a v-on:click="setActive('sites')">Scene data</a></li>
      </ul>
    </div>

    <Folders v-show="active==='folders'"/>
    <Sites v-show="active==='sites'"/>

  </div>
</template>

<script>
  import Folders from "./OptionsFolders.vue";
  import Sites from "./OptionsSites.vue";

  export default {
    components: {Folders, Sites},
    data: function () {
      return {
        active: "folders",
      }
    },
    methods: {
      setActive: function (e) {
        this.active = e;
      }
    }
  }
</script>
