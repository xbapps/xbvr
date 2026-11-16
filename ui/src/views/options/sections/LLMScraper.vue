<template>
  <div class="container">
    <b-loading :is-full-page="false" :active.sync="isLoading" />
    <div class="content">
      <h3>{{ $t('LLM scraper') }}</h3>
      <p>
        Proposes scenes for unmatched files: it searches the web for each filename, reads the pages it
        finds with a language model, and keeps the results as <strong>draft scenes</strong> to review on the
        match screen. Nothing is added to your library until you save a draft.
      </p>
      <hr />
      <b-tabs v-model="activeTab" size="medium" type="is-boxed" style="margin-left: 0px">
        <b-tab-item :label="$t('Language model')"/>
        <b-tab-item :label="$t('Web search')"/>
        <b-tab-item :label="$t('Unmatched files')"/>
      </b-tabs>

      <!-- Language model -->
      <div class="columns" v-if="activeTab == 0">
        <div class="column is-two-thirds">
          <p>Any OpenAI-compatible chat completions endpoint works: vLLM, Ollama, LM Studio, llama.cpp server, or OpenAI itself.</p>
          <b-field :label="$t('Endpoint base URL')" label-position="on-border"
                   message="The part before /chat/completions, e.g. http://localhost:8000/v1 or http://localhost:11434/v1 for Ollama">
            <b-input v-model="baseUrl" placeholder="http://localhost:8000/v1"></b-input>
          </b-field>
          <b-field :label="$t('Model')" label-position="on-border">
            <b-input v-model="model" placeholder="as listed by the endpoint's /v1/models"></b-input>
          </b-field>
          <b-field :label="$t('API key')" label-position="on-border" message="Leave empty for local servers that do not need one.">
            <b-input v-model="apiKey" type="password" password-reveal></b-input>
          </b-field>
          <b-field>
            <b-tooltip :label="$t('Reasoning models can spend their whole token budget thinking and return nothing. This asks the chat template not to think. Turn off if your server rejects unknown request fields.')" :delay="500" type="is-warning" multilined>
              <b-switch v-model="disableThinking">Disable thinking (reasoning models)</b-switch>
            </b-tooltip>
          </b-field>
          <b-field :label="$t('Structured output')" label-position="on-border"
                   message="json_schema constrains the answer while it is generated. Use json_object if your server rejects json_schema.">
            <b-select v-model="structuredOutput">
              <option value="json_schema">json_schema (recommended)</option>
              <option value="json_object">json_object</option>
            </b-select>
          </b-field>
          <b-field grouped>
            <b-field :label="$t('Timeout (seconds)')" label-position="on-border">
              <b-numberinput v-model="timeoutSeconds" :min="10" :max="900" controls-position="compact"/>
            </b-field>
            <b-field :label="$t('Page text sent (characters)')" label-position="on-border">
              <b-numberinput v-model="maxPageChars" :min="1000" :max="100000" :step="1000" controls-position="compact"/>
            </b-field>
          </b-field>
          <b-field grouped>
            <b-button type="is-primary" @click="save" style="margin-right:1em">Save</b-button>
            <b-button @click="test" :loading="testing">Test connection</b-button>
          </b-field>
          <b-notification v-if="testResult" :type="testResult.ok ? 'is-success' : 'is-danger'" :closable="false">
            <span v-if="testResult.ok">Connected to <strong>{{ testResult.model }}</strong> in {{ testResult.latency_ms }} ms.</span>
            <span v-else>{{ testResult.error }}</span>
          </b-notification>
        </div>
      </div>

      <!-- Web search -->
      <div class="columns" v-if="activeTab == 1">
        <div class="column is-two-thirds">
          <b-field :label="$t('Brave Search API key')" label-position="on-border"
                   message="Free plan: about 2,000 searches a month, one per second. Get a key at https://brave.com/search/api/">
            <b-input v-model="braveApiKey" type="password" password-reveal></b-input>
          </b-field>
          <b-field grouped>
            <b-field :label="$t('Pages read per file')" label-position="on-border">
              <b-numberinput v-model="resultsPerFile" :min="1" :max="20" controls-position="compact"/>
            </b-field>
            <b-field :label="$t('Pages read at once')" label-position="on-border">
              <b-numberinput v-model="concurrency" :min="1" :max="8" controls-position="compact"/>
            </b-field>
            <b-field :label="$t('Follow-up rounds')" label-position="on-border"
                     message="How many links from overview or listing pages to follow before judging the last page read (0-5).">
              <b-numberinput v-model="maxFollowRounds" :min="0" :max="5" controls-position="compact"/>
            </b-field>
          </b-field>
          <b-field :label="$t('Highlight drafts from confidence')" label-position="on-border">
            <b-slider v-model="minConfidence" :min="0" :max="1" :step="0.05" :custom-formatter="v => Math.round(v * 100) + '%'"></b-slider>
          </b-field>
          <b-field>
            <b-switch v-model="skipDownloadSites">Skip download and piracy sites</b-switch>
          </b-field>
          <b-field :label="$t('Never read these domains')" label-position="on-border"
                   message="One per line or comma separated. Subdomains are included.">
            <b-input v-model="blockedDomains" type="textarea" rows="3"></b-input>
          </b-field>
          <b-field>
            <b-tooltip :label="$t('Pages are never fetched from private or local network addresses unless this is on, so the scraper cannot be pointed at machines on your network.')" :delay="500" type="is-warning" multilined>
              <b-switch v-model="allowPrivateNetworks">Allow private network addresses</b-switch>
            </b-tooltip>
          </b-field>
          <b-button type="is-primary" @click="save">Save</b-button>

          <h5 style="margin-top:2em">Learned download sites</h5>
          <p>
            Domains the model has classified as download or piracy sites. After {{ learnAfter }} such pages a
            domain is skipped straight from search results, so it no longer uses up a page read. Allow one
            if it was misjudged.
          </p>
          <b-table v-if="learned.length" :data="learned" narrowed>
            <b-table-column field="domain" label="Domain" v-slot="props">
              <a :href="props.row.example_url" target="_blank" rel="noreferrer">{{ props.row.domain }}</a>
            </b-table-column>
            <b-table-column field="count" label="Seen" numeric v-slot="props">{{ props.row.count }}</b-table-column>
            <b-table-column field="blocked" label="Status" v-slot="props">
              <b-tag v-if="props.row.allowed" type="is-success is-light">allowed</b-tag>
              <b-tag v-else-if="props.row.blocked" type="is-danger is-light">skipped</b-tag>
              <b-tag v-else type="is-light">seen once</b-tag>
            </b-table-column>
            <b-table-column v-slot="props">
              <b-button size="is-small" @click="setAllowed(props.row.domain, !props.row.allowed)">
                {{ props.row.allowed ? 'Skip again' : 'Allow' }}
              </b-button>
            </b-table-column>
          </b-table>
          <p v-else class="has-text-grey"><small>None yet.</small></p>
        </div>
      </div>

      <!-- Batch over unmatched files -->
      <div class="columns" v-if="activeTab == 2">
        <div class="column is-two-thirds">
          <p>
            Search for drafts for every unmatched video file in the background. Files that were already
            searched are skipped, so this can be stopped and resumed. Each file uses one web search.
          </p>
          <b-field grouped>
            <b-field :label="$t('Limit (0 = all)')" label-position="on-border">
              <b-numberinput v-model="batchLimit" :min="0" :max="10000" controls-position="compact"/>
            </b-field>
            <b-field>
              <b-checkbox v-model="batchForce">Search again files that were already searched (replaces their unreviewed drafts)</b-checkbox>
            </b-field>
          </b-field>
          <b-field grouped>
            <b-button type="is-primary" @click="startBatch" :disabled="batch.running" style="margin-right:1em">Start</b-button>
            <b-button type="is-danger" outlined @click="stopBatch" :disabled="!batch.running">Stop</b-button>
          </b-field>
          <div v-if="batch.total > 0 || batch.running">
            <b-progress :value="batch.done" :max="batch.total || 1" show-value format="raw" type="is-info">
              {{ batch.done }} / {{ batch.total }} files
            </b-progress>
            <p>
              <strong>{{ batch.drafts_created }}</strong> drafts created,
              <strong>{{ batch.errors }}</strong> errors.
              <span v-if="batch.running && batch.current"><br/>Now: <code>{{ batch.current }}</code></span>
              <span v-if="batch.last_error"><br/>Last error: {{ batch.last_error }}</span>
            </p>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import ky from 'ky'

const field = key => ({
  get () { return this.$store.state.optionsLLMScraper.config[key] },
  set (value) { this.$store.commit('optionsLLMScraper/setField', { key, value }) }
})

export default {
  name: 'LLMScraper',
  data () {
    return {
      activeTab: 0,
      testing: false,
      testResult: null,
      batch: { running: false, total: 0, done: 0, drafts_created: 0, errors: 0 },
      learned: [],
      learnAfter: 2,
      batchLimit: 0,
      batchForce: false,
      batchPoll: null
    }
  },
  mounted () {
    this.$store.dispatch('optionsLLMScraper/load')
    this.refreshBatch()
    this.loadLearned()
  },
  beforeDestroy () {
    if (this.batchPoll) clearInterval(this.batchPoll)
  },
  computed: {
    isLoading () { return this.$store.state.optionsLLMScraper.loading },
    baseUrl: field('baseUrl'),
    model: field('model'),
    apiKey: field('apiKey'),
    disableThinking: field('disableThinking'),
    structuredOutput: field('structuredOutput'),
    timeoutSeconds: field('timeoutSeconds'),
    maxPageChars: field('maxPageChars'),
    braveApiKey: field('braveApiKey'),
    resultsPerFile: field('resultsPerFile'),
    minConfidence: field('minConfidence'),
    concurrency: field('concurrency'),
    maxFollowRounds: field('maxFollowRounds'),
    skipDownloadSites: field('skipDownloadSites'),
    blockedDomains: field('blockedDomains'),
    allowPrivateNetworks: field('allowPrivateNetworks')
  },
  methods: {
    async save () {
      await this.$store.dispatch('optionsLLMScraper/save')
      this.$buefy.toast.open({ message: 'Saved', type: 'is-success' })
    },
    async test () {
      this.testing = true
      this.testResult = null
      try {
        // Test what is on screen, not what was last saved.
        await this.$store.dispatch('optionsLLMScraper/save')
        this.testResult = await ky.post('/api/llmscrape/test', { timeout: 300000 }).json()
      } catch (e) {
        this.testResult = { ok: false, error: e.message }
      } finally {
        this.testing = false
      }
    },
    async startBatch () {
      try {
        const r = await ky.post('/api/llmscrape/batch/start', { json: { force: this.batchForce, limit: this.batchLimit } }).json()
        if (r.error) {
          this.$buefy.toast.open({ message: r.error, type: 'is-danger' })
        }
      } catch (e) {
        let message = e.message
        if (e.response) {
          const body = await e.response.json().catch(() => null)
          if (body && body.error) message = body.error
        }
        this.$buefy.toast.open({ message, type: 'is-danger', duration: 6000 })
      }
      this.refreshBatch()
    },
    async loadLearned () {
      try {
        this.learned = await ky.get('/api/llmscrape/learned-domains').json()
      } catch (e) {
        this.learned = []
      }
    },
    async setAllowed (domain, allowed) {
      this.learned = await ky.post('/api/llmscrape/learned-domains', { json: { domain, allowed } }).json()
    },
    async stopBatch () {
      await ky.post('/api/llmscrape/batch/stop')
      this.refreshBatch()
    },
    async refreshBatch () {
      this.batch = await ky.get('/api/llmscrape/batch/status').json()
      if (this.batch.running && !this.batchPoll) {
        this.batchPoll = setInterval(this.refreshBatch, 3000)
      } else if (!this.batch.running && this.batchPoll) {
        clearInterval(this.batchPoll)
        this.batchPoll = null
      }
    }
  }
}
</script>
