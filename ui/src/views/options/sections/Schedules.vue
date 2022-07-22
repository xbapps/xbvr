<template>
  <div class="container">
    <b-loading :is-full-page="false" :active.sync="isLoading"></b-loading>
    <div class="content">
      <h3>{{$t("Task Schedules")}}</h3>
      <hr/>
      <div class="columns">
        <div class="column">
          <section>
            <b-field>
              <h4>{{$t("Rescrape Schedule")}}</h4>
              <b-switch style="margin-left:2em" v-model="rescrapeEnabled">
                <p>{{ rescrapeEnabled ? 'Enabled' : 'Disabled' }}</p>
              </b-switch>
              <b-switch v-if="rescrapeEnabled" style="margin-left:2em" v-model="useRescrapeTimeRange">
                <p>{{ useRescrapeTimeRange ? 'Use Processing Window' : 'Process All Day' }}</p>
              </b-switch>
            </b-field>
            <b-field v-if="rescrapeEnabled" label="Interval">
              <b-slider v-model="rescrapeHourInterval" :min="1" :max="23" :step="1" ></b-slider>
              <div class="column is-one-third" style="margin-left:.75em">{{`Repeat every ${this.rescrapeHourInterval} hour${this.rescrapeHourInterval > 1 ? 's': ''}`}}</div>
            </b-field>
            <div v-if="useRescrapeTimeRange && rescrapeEnabled">
              <b-field label="Processing Window">
                <b-slider v-model="rescrapeTimeRange" :min="0" :max="48" :step="1" :custom-formatter="val => timeRange[val]" @input="restrictRescrapTo24Hours">
                  <b-slider-tick :value="0">00:00</b-slider-tick>
                  <b-slider-tick :value="6">06:00</b-slider-tick>
                  <b-slider-tick :value="12">12:00</b-slider-tick>
                  <b-slider-tick :value="18">18:00</b-slider-tick>
                  <b-slider-tick :value="24">Midnight</b-slider-tick>
                  <b-slider-tick :value="30">06:00</b-slider-tick>
                  <b-slider-tick :value="36">12:00</b-slider-tick>
                  <b-slider-tick :value="42">18:00</b-slider-tick>
                  <b-slider-tick :value="48">00:00</b-slider-tick>
                </b-slider>
                <div class="column is-one-third" style="margin-left:.75em">{{`${this.timeRange[this.rescrapeTimeRange[0]]} - ${this.timeRange[this.rescrapeTimeRange[1]]}`}}</div>
              </b-field>
              <b-field>
                <b-slider v-model="rescrapeMinuteStart" :min="0" :max="60" :step="1" ></b-slider>
                <div class="column is-one-third" style="margin-left:.75em">{{ minutesStartMsg(rescrapeMinuteStart) }}</div>
              </b-field>
            </div>
            <hr/>
            <b-field>
              <h4>{{$t("Rescan Schedule")}}</h4>
              <b-switch style="margin-left:2em" v-model="rescanEnabled">
                <p>{{ rescanEnabled ? 'Enabled' : 'Disabled' }}</p>
              </b-switch>
              <b-switch v-if="rescanEnabled" style="margin-left:2em" v-model="useRescanTimeRange">
                <p>{{ useRescanTimeRange ? 'Use Processing Window' : 'Process All Day' }}</p>
              </b-switch>
            </b-field>
            <b-field v-if="rescanEnabled" label="Interval">
               <b-slider v-model="rescanHourInterval" :min="1" :max="23" :step="1" ></b-slider>
               <div class="column is-one-third" style="margin-left:.75em">{{`Repeat every ${this.rescanHourInterval} hour${this.rescanHourInterval > 1 ? 's': ''}`}}</div>
            </b-field>
            <div v-if="useRescanTimeRange && rescanEnabled">
              <b-field label="Processing Window">
                <b-slider v-model="rescanTimeRange" :min="0" :max="48" :step="1" :custom-formatter="val => timeRange[val]" @input="restrictRescanTo24Hours">
                  <b-slider-tick :value="0">00:00</b-slider-tick>
                  <b-slider-tick :value="6">06:00</b-slider-tick>
                  <b-slider-tick :value="12">12:00</b-slider-tick>
                  <b-slider-tick :value="18">18:00</b-slider-tick>
                  <b-slider-tick :value="24">Midnight</b-slider-tick>
                  <b-slider-tick :value="30">06:00</b-slider-tick>
                  <b-slider-tick :value="36">12:00</b-slider-tick>
                  <b-slider-tick :value="42">18:00</b-slider-tick>
                  <b-slider-tick :value="48">00:00</b-slider-tick>
                </b-slider>
                <div class="column is-one-third" style="margin-left:.75em">{{`${this.timeRange[this.rescanTimeRange[0]]} - ${this.timeRange[this.rescanTimeRange[1]]}`}}</div>
              </b-field>
              <b-field>
                <b-slider v-model="rescanMinuteStart" :min="0" :max="60" :step="1" ></b-slider>
                <div class="column is-one-third" style="margin-left:.75em">{{ minutesStartMsg(rescanMinuteStart) }}</div>
              </b-field>
            </div>
            <hr/>
            <b-field grouped>
              <b-button type="is-primary" @click="saveSettings" style="margin-right:1em">Save settings</b-button>
            </b-field>
          </section>
          <hr/>
          <section>
            <p>
              NOTE: The processing window cannot be more than 24 hours.
            </p>
            <p>
              Restart XBVR to use new schedule settings
            </p>
          </section>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import ky from 'ky'
import prettyBytes from 'pretty-bytes'

export default {
  name: 'Schedules',
  data () {
    return {
      isLoading: true,
      rescrapeEnabled: true,
      rescanEnabled: true,
      rescrapeTimeRange: [0, 23],
      lastTimeRange: [0, 23],
      rescanTimeRange: [0, 23],
      lastrescanTimeRange: [0, 23],
      useRescrapeTimeRange: false,
      useRescanTimeRange: false,
      rescrapeHourInterval: 0,
      rescrapeMinuteStart: 0,
      rescanMinuteStart: 0,
      rescanHourInterval: 0,
      timeRange: ['00:00', '01:00', '02:00', '03:00', '04:00', '05:00', '06:00', '07:00', '08:00', '09:00', '10:00', '11:00',
        '12:00', '13:00', '14:00', '15:00', '16:00', '17:00', '18:00', '19:00', '20:00', '21:00', '22:00', '23:00',
        '00:00', '01:00', '02:00', '03:00', '04:00', '05:00', '06:00', '07:00', '08:00', '09:00', '10:00', '11:00',
        '12:00', '13:00', '14:00', '15:00', '16:00', '17:00', '18:00', '19:00', '20:00', '21:00', '22:00', '23:00', '00:00']
    }
  },
  async mounted () {
    await this.loadState()
  },
  computed: {
  },
  methods: {
    restrictRescrapTo24Hours () {
      this.rescrapeTimeRange = this.restrictTo24Hours(this.rescrapeTimeRange, this.lastTimeRange)
      this.lastTimeRange = this.rescrapeTimeRange
    },
    restrictRescanTo24Hours () {
      this.rescanTimeRange = this.restrictTo24Hours(this.rescanTimeRange, this.lastrescanTimeRange)
      this.lastrescanTimeRange = this.rescanTimeRange
    },
    restrictTo24Hours (rescrapeTimeRange, lastTimeRange) {
      // check the first time is not in the second 24 hours, no need, should be in the first 24 hours
      if (rescrapeTimeRange[0] > 23) {
        rescrapeTimeRange[0] = 23
        rescrapeTimeRange = [rescrapeTimeRange[0], rescrapeTimeRange[1]]
      }
      // check they are not trying to select more than a 24 hour range
      if ((rescrapeTimeRange[1] - rescrapeTimeRange[0]) > 23 ) {
        if (rescrapeTimeRange[0] === lastTimeRange[0] || rescrapeTimeRange[0] === lastTimeRange[1]) {
          rescrapeTimeRange = [rescrapeTimeRange[1] - 23, rescrapeTimeRange[1]]
        } else {
          rescrapeTimeRange = [rescrapeTimeRange[0], rescrapeTimeRange[0] + 23]
        }
      }
      return rescrapeTimeRange
    },
    async loadState () {
      this.isLoading = true
      await ky.get('/api/options/state')
        .json()
        .then(data => {
          this.rescrapeEnabled = data.config.cron.rescrapeSchedule.enabled
          this.rescrapeHourInterval = data.config.cron.rescrapeSchedule.hourInterval
          this.useRescrapeTimeRange = data.config.cron.rescrapeSchedule.useRange
          this.rescrapeMinuteStart = data.config.cron.rescrapeSchedule.minuteStart
          this.rescanEnabled = data.config.cron.rescanSchedule.enabled
          this.rescanHourInterval = data.config.cron.rescanSchedule.hourInterval
          this.useRescanTimeRange = data.config.cron.rescanSchedule.useRange
          this.rescanMinuteStart = data.config.cron.rescanSchedule.minuteStart
          if (data.config.cron.rescrapeSchedule.hourStart > data.config.cron.rescrapeSchedule.hourEnd) {
            this.rescrapeTimeRange = [data.config.cron.rescrapeSchedule.hourStart, data.config.cron.rescrapeSchedule.hourEnd + 24]
          } else {
            this.rescrapeTimeRange = [data.config.cron.rescrapeSchedule.hourStart, data.config.cron.rescrapeSchedule.hourEnd]
          }
          if (data.config.cron.rescanSchedule.hourStart > data.config.cron.rescanSchedule.hourEnd) {
            this.rescanTimeRange = [data.config.cron.rescanSchedule.hourStart, data.config.cron.rescanSchedule.hourEnd + 24]
          } else {
            this.rescanTimeRange = [data.config.cron.rescanSchedule.hourStart, data.config.cron.rescanSchedule.hourEnd]
          }
          this.isLoading = false
        })
    },
    minutesStartMsg (start) {
      if (start === 0) {
        return 'Start on the hour'
      }
      if (start === 1) {
        return 'Start at 1 minute past the hour'
      }
      return `Start at ${start} minutes past the hour`
    },
    async saveSettings () {
      this.isLoading = true
      await ky.post('/api/options/task-schedule', {
        json: {
          rescrapeEnabled: this.rescrapeEnabled,
          rescrapeHourInterval: this.rescrapeHourInterval,
          rescrapeUseRange: this.useRescrapeTimeRange,
          rescrapeMinuteStart: this.rescrapeMinuteStart,
          rescrapeHourStart: this.rescrapeTimeRange[0],
          rescrapeHourEnd: this.rescrapeTimeRange[1],
          rescanEnabled: this.rescanEnabled,
          rescanHourInterval: this.rescanHourInterval,
          rescanUseRange: this.useRescanTimeRange,
          RescanMinuteStart: this.rescanMinuteStart,
          rescanHourStart: this.rescanTimeRange[0],
          rescanHourEnd: this.rescanTimeRange[1]
        }
      })
        .json()
        .then(data => {
          this.isLoading = false
        })
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
</style>
