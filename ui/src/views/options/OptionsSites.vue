<template>
  <div>
    <div class="columns">
      <div class="column">
        <p>
          Releases metadata is required for XBVR to function as intended.
        </p>
        <hr/>
        <div class="button is-button is-primary" v-on:click="taskScrape()">Run scraper</div>
      </div>
      <div class="column">
        <p>
          Once releases metadata is collected, you should populate search index.
        </p>
        <p>
          Warning: this is CPU-intensive process.
        </p>
        <hr/>
        <div class="button is-button is-primary" v-on:click="taskIndex()">Index scenes</div>
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
    methods: {
      taskScrape() {
        ky.get(`/api/task/scrape`);
      },
      taskIndex() {
        ky.get(`/api/task/index`);
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
