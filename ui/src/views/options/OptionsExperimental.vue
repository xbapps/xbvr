<template>
  <div>
    <div class="columns">
      <div class="column">
        <div class="content">
          <h3 class="title">JAVR via R18 (experimental)</h3>
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
      </div>
      <div class="column">
        <div class="content">
          <h3>Scene search index (experimental)</h3>
          <p>
            Once releases metadata is collected, you should populate search index.<br/>
            This needs to be done whenever new scenes are scraped.
          </p>
          <p>
            Warning: this is CPU-intensive process.
          </p>
          <div class="button is-button is-primary" v-on:click="taskIndex()">Index scenes</div>
          <hr/>
        </div>
      </div>
    </div>
    <div class="columns">
      <div class="column is-full">
        <b-message v-if="Object.keys(lastMessage).length !== 0">
          <span class="icon" v-if="lock">
            <i class="fas fa-spinner fa-pulse"></i>
          </span>
          {{lastMessage.message}}
        </b-message>
      </div>
    </div>
  </div>
</template>

<script>
  import ky from "ky";

  export default {
    name: "OptionsExperimental",
    data() {
      return {
        javrQuery: "",
        bundleURL: "",
      }
    },
    methods: {
      taskIndex() {
        ky.get(`/api/task/index`);
      },
      scrapeJAVR() {
        ky.post(`/api/task/scrape-javr`, {json: {q: this.javrQuery}});
      },
    },
    computed: {
      lastMessage() {
        return this.$store.state.messages.lastScrapeMessage;
      },
      lock() {
        return this.$store.state.messages.lockScrape;
      }
    }
  }
</script>
