<template>
  <div class="container">
    <b-loading :is-full-page="false" :active.sync="isLoading"></b-loading>
    <div class="content">
      <h3>{{$t("Previews")}}</h3>
      <hr/>
      <div class="columns">
        <div class="column">
          <section>
            <b-field label="Snippet length">
              <div class="columns">
                <div class="column is-two-thirds">
                  <b-slider :min="0.2" :max="5" :step="0.1" :tooltip="false" v-model="snippetLength"></b-slider>
                </div>
                <div class="column">
                  <div class="content">{{snippetLength}}sec</div>
                </div>
              </div>
            </b-field>
            <b-field label="Number of snippets">
              <div class="columns">
                <div class="column is-two-thirds">
                  <b-slider :min="2" :max="40" :step="1" :tooltip="false" v-model="snippetAmount"></b-slider>
                </div>
                <div class="column">
                  <div class="content">{{snippetAmount}}</div>
                </div>
              </div>
            </b-field>
            <b-field>
              <b-checkbox v-model="extraSnippet">Grab extra snippet from the end of video</b-checkbox>
            </b-field>
            <b-field>
              <b-checkbox v-model="useCUDA">Use CUDA hardware acceleration</b-checkbox>
            </b-field>
            <b-field label="Preview resolution">
              <div class="columns">
                <div class="column is-two-thirds">
                  <b-slider :min="300" :max="800" :step="20" :tooltip="false" v-model="resolution"></b-slider>
                </div>
                <div class="column">
                  <div class="content">{{resolution}}px</div>
                </div>
              </div>
            </b-field>
            <b-field label="Tilt">
              <div class="columns">
                <div class="column is-two-thirds">
                  <b-slider :min="0" :max="60" :step="1" :tooltip="false" v-model="pitch"></b-slider>
                </div>
                <div class="column">
                  <div class="content">{{pitch}}</div>
                </div>
              </div>
            </b-field>
            <b-field grouped>
              <b-button type="is-primary" @click="saveSettings" style="margin-right:1em">Save settings</b-button>
              <b-button @click="testSettings" style="margin-right:1em">Test settings</b-button>
              <b-button @click="clearTestPreview" v-if="isPreviewReady || previewError || previewElapsed > 0">Clear test preview</b-button>
            </b-field>
          </section>
          <hr/>
          <section>
            <p>
              Once you test preview settings, you can start generating them.<br/>
              When hit Stop - generation process actually stops only after last file finished.
            </p>
            <b-field grouped>
              <b-button type="is-primary" @click="startGenerating" :disabled="previewLeft === 0 || isGenerating" style="margin-right:1em">Start generating previews</b-button>
              <b-button @click="stopPreview" :disabled="!isGenerating">Stop generating</b-button>
            </b-field>
            <p v-if="previewStarted" style="margin-top:0.5em">
              <span v-if="previewCalculating">Calculating...</span>
              <span v-else-if="previewTotal !== null">
                <strong>Total:</strong> {{ previewTotal }} <strong>Left:</strong> {{ previewLeft }}
              </span>
            </p>
          </section>
        </div>
        <div class="column">
          <video v-if="isPreviewReady" :src="`/api/dms/preview/${previewFn}`" autoplay loop></video>
          <div v-if="generatingPreview">
            <div style="display: flex; flex-wrap: wrap;">
              <div class="bbox">
                <b-icon pack="fas" icon="sync-alt" custom-class="fa-spin"></b-icon>
              </div>
            </div>
          </div>
          <div class="timer" v-if="generatingPreview || isPreviewReady">Preview generation time: {{ previewTimer }}</div>
          <b-message v-if="previewError" type="is-danger" :closable="false">
            {{ previewError }}
          </b-message>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import ky from 'ky'
import prettyBytes from 'pretty-bytes'

export default {
  name: 'Previews',
  data () {
    return {
      isLoading: true,
      snippetLength: 0.2,
      snippetAmount: 2,
      resolution: 300,
      extraSnippet: false,
      useCUDA: true,
      pitch: 15,
      timerInterval: null,
      countInterval: null,
      previewLeft: null,
      previewTotal: null,
      previewCalculating: false,
      previewStarted: false
    }
  },
  async mounted () {
    await this.loadState()
    await this.fetchPreviewCount()
    this.countInterval = setInterval(() => {
      this.fetchPreviewCount()
    }, 10000)
  },
  beforeDestroy () {
    this.stopTimer()
    if (this.countInterval) {
      clearInterval(this.countInterval)
      this.countInterval = null
    }
  },
  computed: {
    generatingPreview () {
      return this.$store.state.optionsPreviews.generatingPreview
    },
    isPreviewReady () {
      return this.$store.state.optionsPreviews.isPreviewReady
    },
    previewFn () {
      return this.$store.state.optionsPreviews.previewFn
    },
    previewError () {
      return this.$store.state.optionsPreviews.previewError
    },
    previewElapsed () {
      return this.$store.state.optionsPreviews.previewElapsed
    },
    isGenerating () {
      return this.generatingPreview || this.$store.state.messages.lockPreview
    },
    previewTimer () {
      const total = this.previewElapsed
      const m = Math.floor(total / 60).toString().padStart(2, '0')
      const s = (total % 60).toString().padStart(2, '0')
      return `${m}:${s}`
    }
  },
  watch: {
    generatingPreview (newVal, oldVal) {
      if (newVal && !oldVal) {
        this.startTimer()
      } else if (!newVal && oldVal) {
        this.stopTimer()
      }
    }
  },
  methods: {
    async loadState () {
      this.isLoading = true
      await ky.get('/api/options/state')
        .json()
        .then(data => {
          this.snippetLength = data.config.library.preview.snippetLength
          this.snippetAmount = data.config.library.preview.snippetAmount
          this.resolution = data.config.library.preview.resolution
          this.extraSnippet = data.config.library.preview.extraSnippet
          this.useCUDA = data.config.library.preview.useCUDA !== false
          this.pitch = data.config.library.preview.pitch !== undefined ? data.config.library.preview.pitch : 15
          this.isLoading = false
        })
    },
    async saveSettings () {
      this.isLoading = true
      await ky.put('/api/options/previews', {
        json: {
          snippetLength: this.snippetLength,
          snippetAmount: this.snippetAmount,
          resolution: this.resolution,
          extraSnippet: this.extraSnippet,
          useCUDA: this.useCUDA,
          pitch: this.pitch
        }
      })
        .json()
        .then(data => {
          this.isLoading = false
        })
    },
    async testSettings () {
      this.$store.commit('optionsPreviews/hidePreview')
      await ky.post('/api/options/previews/test', {
        json: {
          snippetLength: this.snippetLength,
          snippetAmount: this.snippetAmount,
          resolution: this.resolution,
          extraSnippet: this.extraSnippet,
          useCUDA: this.useCUDA,
          pitch: this.pitch
        }
      })
    },
    async fetchPreviewCount (setTotal = false) {
      try {
        const data = await ky.get('/api/task/preview/count').json()
        this.previewLeft = data.left
        if (setTotal) {
          this.previewTotal = data.total
        }
      } catch (e) {
        // ignore
      }
    },
    async startGenerating () {
      this.previewStarted = true
      this.previewCalculating = true
      this.previewLeft = null
      this.previewTotal = null
      await ky.get('/api/task/preview/generate')
      await this.fetchPreviewCount(true)
      this.previewCalculating = false
    },
    async stopPreview () {
      await ky.get('/api/task/preview/stop')
    },
    async clearTestPreview () {
      await ky.delete('/api/options/previews/test')
      this.stopTimer()
      this.$store.commit('optionsPreviews/clearPreview')
    },
    startTimer () {
      this.stopTimer()
      this.$store.commit('optionsPreviews/tickPreviewTimer')
      this.timerInterval = setInterval(() => {
        this.$store.commit('optionsPreviews/tickPreviewTimer')
      }, 1000)
    },
    stopTimer () {
      if (this.timerInterval) {
        clearInterval(this.timerInterval)
        this.timerInterval = null
      }
    },
    prettyBytes
  }
}
</script>

<style scoped>
  video {
    width: 100%;
  }

  .bbox {
    flex: 1 0 calc(25% - 10px);
    margin: 5px;
    background: #f0f0f0;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .bbox:after {
    content: '';
    display: block;
    padding-bottom: 100%;
  }

  .timer {
    text-align: center;
    font-size: 1.5em;
    font-weight: bold;
    margin-top: 0.5em;
    font-family: monospace;
  }
</style>
