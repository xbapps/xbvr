<template>
  <div class="container">
    <b-loading :is-full-page="false" :active.sync="isLoading" />
    <div class="content">
      <h3>{{ $t('Recommendations') }}</h3>
      <p>
        Builds two deo-enabled playlists from your watch history, ratings and favourites:
        <strong>For You</strong> (scenes to watch) and <strong>Cleanup</strong> (scenes you're
        least likely to miss). Adjust below, then recompute.
      </p>
      <hr />

      <div class="columns">
        <div class="column">
          <b-field>
            <b-switch v-model="enabled" type="is-success">Enable recommendations</b-switch>
          </b-field>
          <b-field>
            <b-switch v-model="useLearnedModel" type="is-link">
              Learn taste automatically (trains on your ratings/favourites/watch history;
              falls back to the weights below until there's enough signal)
            </b-switch>
          </b-field>
          <b-field v-if="useLearnedModel" label="Learner">
            <b-select v-model="modelType">
              <option value="linear">Linear (logistic regression)</option>
              <option value="fm">Factorization machine (learns feature interactions)</option>
            </b-select>
          </b-field>
          <b-field v-if="useLearnedModel">
            <b-switch v-model="useVisualEmbeddings" type="is-link">
              Learn visual taste (CNN embeds one frame/scene as model features; first run
              embeds your whole library in the background, then it's cached)
            </b-switch>
          </b-field>
          <b-field>
            <b-switch v-model="excludeRecentlyWatched" type="is-dark">Don’t consider recently-watched scenes in “For You”</b-switch>
          </b-field>

          <b-field label="“For You” list size">
            <b-numberinput v-model="watchListSize" :min="0" :max="1000" :step="10" controls-position="compact" />
          </b-field>
          <b-field label="“Cleanup” list size">
            <b-numberinput v-model="deleteListSize" :min="0" :max="1000" :step="10" controls-position="compact" />
          </b-field>

          <b-field label="Diversity (1.0 = repeat favourites freely, lower = more variety)">
            <b-numberinput v-model="diversityDecay" :min="0" :max="1" :step="0.05" controls-position="compact" />
          </b-field>
          <b-field label="Protect rating ≥ (never suggest deleting at/above this star rating)">
            <b-numberinput v-model="protectRating" :min="0" :max="5" :step="0.5" controls-position="compact" />
          </b-field>
          <b-field label="Grace period (days) before a newly-added scene can be a cleanup candidate">
            <b-numberinput v-model="graceDays" :min="0" :max="365" :step="1" controls-position="compact" />
          </b-field>
        </div>

        <div class="column">
          <b-field label="Weights">
          </b-field>
          <b-field label="Actor affinity">
            <b-numberinput v-model="wActor" :min="0" :max="3" :step="0.1" controls-position="compact" />
          </b-field>
          <b-field label="Tag affinity">
            <b-numberinput v-model="wTag" :min="0" :max="3" :step="0.1" controls-position="compact" />
          </b-field>
          <b-field label="Site affinity">
            <b-numberinput v-model="wSite" :min="0" :max="3" :step="0.1" controls-position="compact" />
          </b-field>
          <b-field label="Quality (resolution)">
            <b-numberinput v-model="wQuality" :min="0" :max="3" :step="0.1" controls-position="compact" />
          </b-field>
          <b-field label="Freshness (recency)">
            <b-numberinput v-model="wFreshness" :min="0" :max="3" :step="0.1" controls-position="compact" />
          </b-field>
          <b-field label="Cleanup size bias (reclaim larger files first)">
            <b-numberinput v-model="wSize" :min="0" :max="3" :step="0.1" controls-position="compact" />
          </b-field>

          <b-field label="Visual quality re-rank (0 = off; samples frames of the listed files)">
            <b-numberinput v-model="wVisualQuality" :min="0" :max="3" :step="0.1" controls-position="compact" />
          </b-field>
          <b-field label="Quality frames sampled per file (1 every 5 min; higher = slower)">
            <b-numberinput v-model="vqMaxSamples" :min="1" :max="24" :step="1" controls-position="compact" />
          </b-field>
          <b-field label="Daily variety (0 = deterministic; higher shuffles the lists more each day)">
            <b-numberinput v-model="noiseWeight" :min="0" :max="1" :step="0.05" controls-position="compact" />
          </b-field>
        </div>
      </div>

      <hr />
      <div class="buttons">
        <b-button type="is-primary" @click="save">Save settings</b-button>
        <b-button type="is-link" @click="recompute" :loading="isLoading">Recompute now</b-button>
      </div>
    </div>
  </div>
</template>

<script>
const FIELDS = [
  'enabled', 'useLearnedModel', 'modelType', 'useVisualEmbeddings', 'excludeRecentlyWatched', 'watchListSize', 'deleteListSize', 'diversityDecay',
  'protectRating', 'graceDays', 'wActor', 'wTag', 'wSite', 'wQuality', 'wFreshness', 'wSize',
  'wVisualQuality', 'vqMaxSamples', 'noiseWeight'
]

export default {
  name: 'Recommendations',
  computed: {
    isLoading () {
      return this.$store.state.optionsRecommendations.loading
    },
    ...FIELDS.reduce((acc, key) => {
      acc[key] = {
        get () { return this.$store.state.optionsRecommendations.config[key] },
        set (value) { this.$store.commit('optionsRecommendations/setField', { key, value }) }
      }
      return acc
    }, {})
  },
  methods: {
    save () {
      this.$store.dispatch('optionsRecommendations/save').then(() => {
        this.$buefy.toast.open({ message: 'Recommendation settings saved', type: 'is-success' })
      })
    },
    recompute () {
      this.$store.dispatch('optionsRecommendations/recompute').then(() => {
        this.$buefy.toast.open({ message: 'Recomputing recommendations in the background…', type: 'is-info' })
      })
    }
  },
  mounted () {
    this.$store.dispatch('optionsRecommendations/load')
  }
}
</script>
