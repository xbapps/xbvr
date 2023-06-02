<template>
  <div class="container">
    <b-loading :is-full-page="false" :active.sync="isLoading" />
    <div class="content">
      <h3>{{ $t('Advanced') }}</h3>
      <hr />
      <b-tabs v-model="activeTab" size="medium" type="is-boxed" style="margin-left: 0px" id="importexporttab">
            <b-tab-item label="Scene Details"/>
            <b-tab-item label="Create Custom Site"/>
      </b-tabs>

      <!-- Screen Details Tab -->
      <div class="columns" v-if="activeTab == 0">
        <div class="column">
          <section>
            <b-field>
              <b-switch v-model="showInternalSceneId" type="is-default">
                show Internal Scene Id
              </b-switch>
            </b-field>
            <b-field>
              <b-switch v-model="showHSPApiLink" type="is-default">
                show Heresphere Api Link
              </b-switch>
            </b-field>

            <b-field>
              <b-button type="is-primary" @click="save">Save</b-button>
            </b-field>
          </section>
        </div>
      </div>
      
      <!-- Custom Sites Tab -->
      <div class="columns" v-if="activeTab == 1">
        <div class="column">
          <section>
            <b-field :label="$t('Scraper Url')" label-position="on-border">
              <b-input v-model="scraperUrl" :placeholder="$t('Enter the Url to Studio Scene List')" @input="validateScraperFields()"></b-input>
            </b-field>
            <b-field :label="$t('Name')" label-position="on-border">
              <b-input v-model="scraperName" :placeholder="$t('Enter Studio Name')" @input="validateScraperFields()"></b-input>
            </b-field>
            <b-field :label="$t('Company')" label-position="on-border">
              <b-input v-model="scraperCompany" :placeholder="$t('Optional: defaults to Name')"></b-input>
            </b-field>
            <b-field :label="$t('Avatar Url')" label-position="on-border">
              <b-input v-model="scraperAvatar" :placeholder="$t('Optional')"></b-input>
            </b-field>  
            <b-tooltip :label="$t('Restart XBR to load new Sites')" :delay="500" type="is-warning">
              <b-field>
                <b-button type="is-primary" :disabled="!scraperFieldsValid" @click="saveScraper">Save</b-button>
              </b-field>
            </b-tooltip>
          </section>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import ky from 'ky'
export default {
  name: 'InterfaceAdvanced',
  mounted () {    
    this.$store.dispatch('optionsAdvanced/load')    
  },
  data () {
    return {
      activeTab: 0,
      scraperUrl: '',
      scraperName: '',
      scraperCompany: '',
      scraperAvatar: '',
      scraperFieldsValid: false,
      }
  },
  methods: {
    save () {
      this.$store.dispatch('optionsAdvanced/save')
    },
    validateScraperFields() {
      console.log("validateScraperFields")
      this.scraperFieldsValid=false
      if (this.scraperName != "") {
        if (this.scraperUrl.startsWith("https://") || this.scraperUrl.startsWith("http://") ) {
          if (this.scraperUrl.includes("povr.com") || this.scraperUrl.includes("sexlikereal.com") || this.scraperUrl.includes("vrphub.com") || this.scraperUrl.includes("vrporn.com")) {
            this.scraperFieldsValid=true
          }
        }
      }
    },
    saveScraper () {
       ky.put('/api/options/custom-sites/create', {
        json: {
          scraperUrl: this.scraperUrl,
          scraperName: this.scraperName,
          scraperCompany: this.scraperCompany,
          scraperAvatar: this.scraperAvatar
        }
      })

    },
  },
  computed: {
    showInternalSceneId: {
      get () {
        return this.$store.state.optionsAdvanced.advanced.showInternalSceneId
      },
      set (value) {
        this.$store.state.optionsAdvanced.advanced.showInternalSceneId = value
      }
    },
    showHSPApiLink: {
      get () {
        return this.$store.state.optionsAdvanced.advanced.showHSPApiLink
      },
      set (value) {
        this.$store.state.optionsAdvanced.advanced.showHSPApiLink = value
      }
    },    
    isLoading: function () {
      return this.$store.state.optionsAdvanced.loading
    }
  }
}
</script>

<style scoped>

</style>
