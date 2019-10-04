<template>
  <div class="container">
    <div class="tabs is-medium is-boxed">
      <ul>
        <li v-bind:class="{'is-active': active==='folders'}"><a v-on:click="setActive('folders')">Folders</a></li>
        <li v-bind:class="{'is-active': active==='sites'}"><a v-on:click="setActive('sites')">Scene data</a></li>
        <li v-bind:class="{'is-active': active==='experimental'}"><a v-on:click="setActive('experimental')">Experimental</a></li>
      </ul>
    </div>

    <Folders v-show="active==='folders'"/>
    <Sites v-show="active==='sites'"/>
    <Experimental v-show="active==='experimental'"/>

  </div>
</template>

<script>
  import Folders from "./OptionsFolders.vue";
  import Sites from "./OptionsSites.vue";
  import Experimental from "./OptionsExperimental.vue";

  export default {
    components: {Folders, Sites, Experimental},
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
