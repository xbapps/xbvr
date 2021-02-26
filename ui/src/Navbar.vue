<template>
  <b-navbar :fixed-top="true" type="is-light">
    <template slot="brand">
      <b-navbar-item>
        <h1 class="title">XBVR <small>{{currentVersion}}</small></h1>
      </b-navbar-item>
    </template>
    <template slot="start">
      <b-navbar-item tag="router-link" :to="{ path: './' }">
        {{$t('Scenes')}}
      </b-navbar-item>
      <b-navbar-item tag="router-link" :to="{ path: './files' }">
        {{$t('Files')}}
      </b-navbar-item>
      <b-navbar-item tag="router-link" :to="{ path: './options' }">
        {{$t('Options')}}
      </b-navbar-item>
      <b-navbar-item @click="$store.commit('overlay/showQuickFind')">
        {{$t('Quick find')}}
      </b-navbar-item>
    </template>
    <template slot="end">
      <b-navbar-item>
        <table style="font-size:0.9em">
          <tr v-if="Object.keys(lastRescanMessage).length !== 0">
            <th><span :class="[lockRescan ? 'pulsate' : '']">{{$t('Files')}} →</span></th>
            <td>{{lastRescanMessage.message}}</td>
          </tr>
          <tr v-if="Object.keys(lastScrapeMessage).length !== 0">
            <th><span :class="[lockScrape ? 'pulsate' : '']">{{$t('Data')}} →</span></th>
            <td>{{lastScrapeMessage.message}}</td>
          </tr>
        </table>
      </b-navbar-item>
    </template>
  </b-navbar>
</template>

<script>
import ky from 'ky'

export default {
  data () {
    return {
      currentVersion: '',
      latestVersion: ''
    }
  },
  computed: {
    lockRescan () {
      return this.$store.state.messages.lockRescan
    },
    lastRescanMessage () {
      return this.$store.state.messages.lastRescanMessage
    },
    lockScrape () {
      return this.$store.state.messages.lockScrape
    },
    lastScrapeMessage () {
      return this.$store.state.messages.lastScrapeMessage
    }
  },
  mounted () {
    ky.get('/api/options/version-check').json().then(data => {
      this.currentVersion = data.current_version
      this.latestVersion = data.latest_version

      if (data.update_notify && this.currentVersion !== 'CURRENT') {
        this.$buefy.snackbar.open({
          message: `Version ${this.latestVersion} available!`,
          type: 'is-warning',
          position: 'is-top',
          actionText: this.$t('Download now'),
          indefinite: true,
          onAction: () => {
            window.location = 'https://github.com/xbapps/xbvr/releases'
          }
        })
      }
    })
  }
}
</script>

<style scoped>
  h1 {
    display: flex;
    align-items: center;
  }

  h1 small {
    font-size: 0.5em;
    margin-left: 0.5em;
    opacity: 0.5;
  }

  th {
    padding-right: 1em;
  }

  .pulsate {
    -webkit-animation: pulsate 0.5s linear;
    -webkit-animation-iteration-count: infinite;
    opacity: 0.5;
  }

  @-webkit-keyframes pulsate {
    0% {
      opacity: 0.5;
    }
    50% {
      opacity: 1.0;
    }
    100% {
      opacity: 0.5;
    }
  }
</style>
