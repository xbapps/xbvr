<template>
  <div class="content">
    <h3 class="title">JAVR scraping via R18</h3>
    <p>
      You can scrape JAVR releases by using:
    </p>
    <ul>
      <li>R18 URL to the exact scene (preferred method)</li>
      <li>production code (XXXX-001)</li>
    </ul>
    <p>
      <b-field grouped>
        <b-input v-model="javrQuery" placeholder="URL or ID" type="search" icon="magnify"></b-input>
        <div class="button is-button is-primary" v-on:click="scrapeJAVR()">Get release</div>
      </b-field>
    </p>
  </div>
</template>

<script>
  import ky from "ky";

  export default {
    name: "OptionsSceneDataJAVR",
    data() {
      return {
        javrQuery: "",
      }
    },
    methods: {
      scrapeJAVR() {
        ky.post(`/api/task/scrape-javr`, {json: {q: this.javrQuery}});
      },
    }
  }
</script>
