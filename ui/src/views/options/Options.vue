<template>
  <div class="container">
    <div class="tabs is-medium is-boxed">
      <ul>
        <li v-bind:class="{'is-active': active==='folders'}"><a v-on:click="setActive('folders')">Folders</a></li>
        <li v-bind:class="{'is-active': active==='scrapers'}"><a v-on:click="setActive('scrapers')">Scrapers</a></li>
        <li v-bind:class="{'is-active': active==='experimental'}"><a v-on:click="setActive('experimental')">Experimental</a></li>
      </ul>
    </div>

    <Folders v-show="active==='folders'"/>
    <Scrapers v-show="active==='scrapers'"/>
    <Experimental v-show="active==='experimental'"/>

  </div>
</template>

<script>
  import Folders from "./OptionsFolders.vue";
  import Scrapers from "./OptionsScrapers.vue";
  import Experimental from "./OptionsExperimental.vue";

  export default {
    components: {Folders, Scrapers, Experimental},
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
