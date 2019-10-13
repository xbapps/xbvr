<template>
  <div>
    <div class="columns">
      <div class="column">
        <b-table :data="items" ref="table"
                 paginated per-page="12" pagination-position="both" default-sort="name">
          <template slot-scope="props">
            <b-table-column field="is_enabled" label="" width="20">
              <b-switch :value="props.row.is_enabled"
                        @input="$store.dispatch('optionsSites/toggleSite', {id: props.row.id})"/>
            </b-table-column>
            <b-table-column field="name" label="Site" sortable>
              {{props.row.name}}
            </b-table-column>
            <b-table-column field="last_update" label="Last update" sortable>
              <span v-if="runningScrapers.includes(props.row.id)">
                <b-progress type="is-primary"></b-progress>
              </span>
              <span v-else-if="props.row.last_update !== '0001-01-01T00:00:00Z'">
                {{formatDistanceToNow(parseISO(props.row.last_update))}} ago
              </span>
              <span v-else>Never</span>
            </b-table-column>
          </template>
          <template slot="top-left">
            <div class="buttons">
              <a class="button is-primary" v-on:click="taskScrape()">Run Selected Scrapers</a>
              <a class="button is-primary" v-on:click="taskScrapeAll()">Run All Scrapers</a>
            </div>
          </template>
        </b-table>
      </div>
      <div class="column">
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
  import {formatDistanceToNow, parseISO} from "date-fns";

  export default {
    name: "OptionsSites",
    data() {
      return {
        bundleURL: "",
      }
    },
    mounted() {
      this.$store.dispatch("optionsSites/load");
    },
    methods: {
      taskScrape() {
        ky.get(`/api/task/scrape`);
      },
      taskScrapeAll() {
        ky.get(`/api/task/scrape/all`);
      },
      importContent() {
        if (this.bundleURL !== "") {
          ky.get(`/api/task/bundle/import`, {searchParams: {url: this.bundleURL}});
        }
      },
      exportContent() {
        ky.get(`/api/task/bundle/export`);
      },
      parseISO,
      formatDistanceToNow,
    },
    computed: {
      items() {
        return this.$store.state.optionsSites.items;
      },
      lastMessage() {
        return this.$store.state.messages.lastScrapeMessage;
      },
      lock() {
        return this.$store.state.messages.lockScrape;
      },
      runningScrapers() {
        this.$store.dispatch("optionsSites/load");
        return this.$store.state.messages.runningScrapers;
      }
    }
  }
</script>
