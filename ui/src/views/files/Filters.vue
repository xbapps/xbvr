<template>
  <div class="field">
    <section>
      <div class="columns">
        <div class="column">
          <label class="label">{{$t("State")}}</label>
          <b-field>
            <b-radio-button v-model="fileState" native-value="all">
              <span>{{$t("All")}}</span>
            </b-radio-button>
            <b-radio-button v-model="fileState" native-value="matched">
              <span>{{$t("Matched")}}</span>
            </b-radio-button>
            <b-radio-button v-model="fileState" native-value="unmatched">
              <span>{{$t("Unmatched")}}</span>
            </b-radio-button>
          </b-field>
        </div>
        <div class="column is-one-fifth">
          <label class="label">{{$t("Filename")}}</label>
          <b-field>
            <b-input v-model="fileName"></b-input>
            <button class="button is-light" @click="clearFilename">
              <b-icon pack="fas" icon="times" size="is-small"></b-icon>
            </button>
          </b-field>
        </div>
        <div class="column is-one-fifth">
          <label class="label">{{$t("Created between")}}</label>
          <b-field>
            <b-datepicker v-model="fileCreation" editable range>
              <div class="buttons">
                <b-button size="is-small" @click="setRange(subDays(new Date(), 7), new Date())">
                  <span>{{$t("Last 7 days")}}</span>
                </b-button>
                <b-button size="is-small" @click="setRange(subDays(new Date(), 14), new Date())">
                  <span>{{$t("Last 14 days")}}</span>
                </b-button>
                <b-button size="is-small" @click="setRange(subDays(new Date(), 30), new Date())">
                  <span>{{$t("Last 30 days")}}</span>
                </b-button>
              </div>
            </b-datepicker>
            <button class="button is-light" @click="clearRange">
              <b-icon pack="fas" icon="times" size="is-small"></b-icon>
            </button>
          </b-field>
        </div>
        <div class="column">
          <label class="label">{{$t("Resolution")}}</label>
          <b-dropdown v-model="fileResolutions" multiple hoverable aria-role="list">
              <button class="button" type="button" slot="trigger">
                  <span>{{$t("Selected")}} ({{fileResolutions.length}})</span>
                  <b-icon icon="menu-down"></b-icon>
              </button>
              <b-dropdown-item value="below4k" aria-role="listitem">
                  <span>{{$t("Below 4K")}}</span>
              </b-dropdown-item>
              <b-dropdown-item value="4k" aria-role="listitem">
                  <span>4K</span>
              </b-dropdown-item>
              <b-dropdown-item value="5k" aria-role="listitem">
                  <span>5K</span>
              </b-dropdown-item>
              <b-dropdown-item value="6k" aria-role="listitem">
                  <span>6K</span>
              </b-dropdown-item>
              <b-dropdown-item value="above6k" aria-role="listitem">
                  <span>{{$t("Above 6K")}}</span>
              </b-dropdown-item>
          </b-dropdown>
        </div>
        <div class="column">
          <label class="label">{{$t("Bitrate")}}</label>
          <b-dropdown v-model="fileBitrates" multiple hoverable aria-role="list">
              <button class="button" type="button" slot="trigger">
                  <span>{{$t("Selected")}} ({{fileBitrates.length}})</span>
                  <b-icon icon="menu-down"></b-icon>
              </button>
              <b-dropdown-item value="low" aria-role="listitem">
                  <span>{{$t("Low (below 15 Mbps)")}}</span>
              </b-dropdown-item>
              <b-dropdown-item value="medium" aria-role="listitem">
                  <span>{{$t("Medium (15 to 24 Mbps)")}}</span>
              </b-dropdown-item>
              <b-dropdown-item value="high" aria-role="listitem">
                  <span>{{$t("High (25 to 35 Mbps)")}}</span>
              </b-dropdown-item>
              <b-dropdown-item value="ultra" aria-role="listitem">
                  <span>{{$t("Ultra (above 35 Mbps)")}}</span>
              </b-dropdown-item>
          </b-dropdown>
        </div>
        <div class="column">
          <label class="label">{{$t("Framerate")}}</label>
          <b-dropdown v-model="fileFramerates" multiple hoverable aria-role="list">
              <button class="button" type="button" slot="trigger">
                  <span>{{$t("Selected")}} ({{fileFramerates.length}})</span>
                  <b-icon icon="menu-down"></b-icon>
              </button>
              <b-dropdown-item value="30fps" aria-role="listitem">
                  <span>30</span>
              </b-dropdown-item>
              <b-dropdown-item value="60fps" aria-role="listitem">
                  <span>60</span>
              </b-dropdown-item>
              <b-dropdown-item value="other" aria-role="listitem">
                  <span>{{$t("Other")}}</span>
              </b-dropdown-item>
          </b-dropdown>
        </div>
      </div>
    </section>
  </div>
</template>

<script>
import { subDays } from 'date-fns'

export default {
  name: 'Filters',
  methods: {
    clearFilename () {
      this.fileName = ''
    },
    clearRange () {
      this.fileCreation = []
    },
    setRange (start, end) {
      this.fileCreation = [start, end]
    },
    subDays
  },
  computed: {
    fileName: {
      get () {
        return this.$store.state.files.filters.filename
      },
      set (value) {
        this.$store.state.files.filters.filename = value
        if (value.length > 3 || value.length == 0) {
          this.$store.dispatch('files/load')
        }
      }
    },
    fileBitrates: {
      get () {
        return this.$store.state.files.filters.bitrates
      },
      set (values) {
        this.$store.state.files.filters.bitrates = values
        this.$store.dispatch('files/load')
      }
    },
    fileFramerates: {
      get () {
        return this.$store.state.files.filters.framerates
      },
      set (values) {
        this.$store.state.files.filters.framerates = values
        this.$store.dispatch('files/load')
      }
    },
    fileResolutions: {
      get () {
        return this.$store.state.files.filters.resolutions
      },
      set (values) {
        this.$store.state.files.filters.resolutions = values
        this.$store.dispatch('files/load')
      }
    },
    fileState: {
      get () {
        return this.$store.state.files.filters.state
      },
      set (value) {
        this.$store.state.files.filters.state = value
        this.$store.dispatch('files/load')
      }
    },
    fileCreation: {
      get () {
        return this.$store.state.files.filters.createdDate
      },
      set (value) {
        this.$store.state.files.filters.createdDate = value
        this.$store.dispatch('files/load')
      }
    }
  }
}
</script>
