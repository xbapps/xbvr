<template>
  <div>
    <div class="columns">
      <div class="column">
        <div class="content">
          <h3>Mainstream sites</h3>
          <p>
            Releases metadata is required for XBVR to function as intended.
          </p>
          <div class="button is-button is-primary" v-on:click="taskScrape()">Run scraper</div>
          <hr/>
        </div>
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
        <div class="content">
          <h3>Import scene data</h3>
          <p>
            You can import existing content bundles in JSON format from URL.
          </p>
          <b-field grouped>
            <b-input v-model="bundleURL" placeholder="Bundle URL" type="search" icon="web"></b-input>
            <div class="button is-button is-primary" v-on:click="importContent">Import content bundle</div>
          </b-field>
          <hr/>
        </div>
        <div class="content">
          <h3>Export scene data</h3>
          <p>
            If you already have scraped scene data, you can export it below.
          </p>
          <b-button type="is-primary" @click="exportContent">Export content bundle</b-button>
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
    name: "OptionsSites.vue",
    data() {
      return {
        javrQuery: "",
        bundleURL: "",
      }
    },
    methods: {
      taskScrape() {
        ky.get(`/api/task/scrape`);
      },
      taskIndex() {
        ky.get(`/api/task/index`);
      },
      scrapeJAVR() {
        ky.post(`/api/task/scrape-javr`, {json: {q: this.javrQuery}});
      },
      importContent() {
        if (this.bundleURL !== "") {
          ky.get(`/api/task/bundle/import`, {searchParams: {url: this.bundleURL}});
        }
      },
      exportContent() {
        ky.get(`/api/task/bundle/export`);
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
