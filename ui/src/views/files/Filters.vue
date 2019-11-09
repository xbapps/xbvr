<template>
  <div class="field">
    <section> 
      <div class="columns">
        <div class="column">
          <label class="label">State</label>
          <b-field>
              <b-radio-button v-model="fileStatus" native-value="all">
                  <span>All</span>
              </b-radio-button>
              <b-radio-button v-model="fileStatus" native-value="matched">
                  <span>Matched</span>
              </b-radio-button>
              <b-radio-button v-model="fileStatus" native-value="unmatched">
                  Unmatched
              </b-radio-button>
          </b-field>
        </div>
        <div class="column">
          <label class="label">Created When</label>
          <b-field>
            <b-datepicker v-model="fileCreation" multiple>
            </b-datepicker>
          </b-field>
        </div>
        <div class="column">
          <label class="label">Height</label>
          <b-field>
              <b-slider v-model="height" :min="900" :max="9000" :step="1000" ticks>
              </b-slider>
          </b-field>
        </div>
        <div class="column">
          <label class="label">Bitrate</label>
          <b-field>
              <b-slider v-model="bitrate" :min="0" :max="50" :step="5" ticks>
              </b-slider>
          </b-field>
        </div>
        <div class="column">
          <label class="label">Framerate</label>
          <b-field>
              <b-slider v-model="framerate" :min="0" :max="90" :step="10" ticks>
              </b-slider>
          </b-field>
        </div>
      </div>  
    </section>
  </div>
</template>

<script>
  export default {
    name: "Filters",
    data() {
      return {
        fileStatus: 'all',
        height: [900, 3000],
        bitrate: [20, 30],
        framerate: [30,40],
        fileCreation: [],
      }
    },
    methods: {
    },
    computed: {
    }
  }
</script>
