<template>
  <div class="content">
    <h3 class="title">Mainstream scrapers</h3>
    <div class="buttons">
      <a class="button is-primary" v-on:click="taskScrape()">Run Selected Scrapers</a>
      <a class="button is-primary" v-on:click="taskScrapeAll()">Run All Scrapers</a>
    </div>
    <b-table :data="items" ref="table" default-sort="name">
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
          <span v-else>never</span>
        </b-table-column>
      </template>
    </b-table>
  </div>
</template>

<script>
  import ky from "ky";
  import {formatDistanceToNow, parseISO} from "date-fns";

  export default {
    name: "OptionsSites",
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
      parseISO,
      formatDistanceToNow,
    },
    computed: {
      items() {
        return this.$store.state.optionsSites.items;
      },
      runningScrapers() {
        this.$store.dispatch("optionsSites/load");
        return this.$store.state.messages.runningScrapers;
      }
    }
  }
</script>
