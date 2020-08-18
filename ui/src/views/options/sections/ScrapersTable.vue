<template>
  <div class="content">
    <div class="columns title-columns">
      <div class="column">
        <h3 class="title">{{$t('Mainstream sites')}}</h3>
      </div>
      <div class="column buttons">
        <a class="button is-primary" @click="taskScrape('_enabled')">{{$t('Run selected scrapers')}}</a>
      </div>
    </div>

    <b-table
      class="scraper-table"
      :data="this.items"
      :sticky-header="true"
      default-sort-direction="asc"
      sort-icon="arrow-up"
      sort-icon-size="is-small"
      default-sort="name">

      <b-table-column field="is_enabled" label="" width="40" centered v-slot="props">
        <b-switch :value="props.row.is_enabled" @input="$store.dispatch('optionsSites/toggleSite', {id: props.row.id})" />
      </b-table-column>

      <b-table-column field="avatar_url" label="" width="50" centered v-slot="props">
        <vue-load-image>
          <img slot="image" :src="getImageURL(props.row.avatar_url || '/ui/images/blank.png')" />
          <img slot="preloader" src="/ui/images/blank.png" />
          <img slot="error" src="/ui/images/blank.png" />
        </vue-load-image>
      </b-table-column>

      <b-table-column field="name" label="Name" searchable sortable v-slot="props">
        {{ props.row.name }}
      </b-table-column>

      <!-- Studio/Scraper used -->
      <b-table-column field="studio" label="Studio" searchable sortable v-slot="props">
        {{ props.row.studio }}
      </b-table-column>

      <b-table-column field="last_update" label="Last Update" sortable v-slot="props">
        <template v-if="runningScrapers.includes(props.row.id)">
          {{ $t('Scraping now...') }}
        </template>
        <template v-else>
          <template v-if="props.row.last_update !== '0001-01-01T00:00:00Z'">
            {{ formatDistanceToNow(parseISO(props.row.last_update)) }}
          </template>
          <template v-else>
            {{ $t('Never scraped') }}
          </template>
        </template>
      </b-table-column>

      <b-table-column v-slot="props">
        <div class="menu">
          <b-dropdown aria-role="list" class="is-pulled-right" position="is-bottom-left">
            <template slot="trigger">
              <b-icon icon="dots-vertical" />
            </template>
            <b-dropdown-item aria-role="listitem" @click="taskScrape(props.row.id)">
              {{ $t('Scrape this site') }}
            </b-dropdown-item>
            <b-dropdown-item aria-role="listitem" @click="forceSiteUpdate(props.row.name)">
              {{ $t('Force update scenes') }}
            </b-dropdown-item>
            <b-dropdown-item aria-role="listitem" @click="deleteScenes(props.row.name)">
              {{ $t('Delete scraped scenes') }}
            </b-dropdown-item>
          </b-dropdown>
        </div>
      </b-table-column>
    </b-table>
  </div>
</template>

<script>
  import ky from "ky";
  import VueLoadImage from "vue-load-image";
  import {formatDistanceToNow, parseISO} from "date-fns";

  export default {
    name: "ScrapersTable",
    components: {VueLoadImage},
    mounted() {
      this.$store.dispatch("optionsSites/load");
    },
    methods: {
      getImageURL(u) {
        return u.startsWith("http")
          ? "/img/128x/" + u.replace("://", ":/")
          : u;
      },
      taskScrape(site) {
        ky.get(`/api/task/scrape?site=${site}`);
      },
      forceSiteUpdate(site) {
        ky.post(`/api/options/scraper/force-site-update`, {
          json: {"site_name": site},
        });
        this.$buefy.toast.open(`Scenes from ${site} will be updated on the next scrape`);
      },
      deleteScenes(site) {
        this.$buefy.dialog.confirm({
          title: this.$t('Delete scraped scenes'),
          message: `You're about to delete scraped scenes for <strong>${site}</strong>. Previously matched file will return to an unmatched state.`,
          type: 'is-danger',
          hasIcon: true,
          onConfirm: () => ky.post(`/api/options/scraper/delete-scenes`, {
            json: {"site_name": site},
          }),
        });
      },
      parseISO,
      formatDistanceToNow,
    },
    computed: {
      items() {
        return this.$store.state.optionsSites.items;
      },
      enabledItems() {
        return this.items.filter(s => s.is_enabled);
      },
      runningScrapers() {
        this.$store.dispatch("optionsSites/load");
        return this.$store.state.messages.runningScrapers;
      }
    }
  }
</script>

<style scoped>
  .title-columns .buttons {
    text-align: right;
  }

  .vue-load-image img {
    max-width: 50px;
  }
</style>

<style>
  .scraper-table > .table-wrapper {
    max-height: 60vh;
    height: 60vh !important;
  }
</style>
