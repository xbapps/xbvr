<template>
  <b-navbar :fixed-top="true" type="is-light">
    <template slot="brand">
      <b-navbar-item>
        <h1 class="title">XBVR <small>{{currentVersion}}</small></h1>
      </b-navbar-item>
    </template>
    <template slot="start">
      <b-navbar-item>
        <router-link to="./">Scenes</router-link>
      </b-navbar-item>
      <b-navbar-item>
        <router-link to="./files">Files</router-link>
      </b-navbar-item>
      <b-navbar-item>
        <router-link to="./options">Options</router-link>
      </b-navbar-item>
    </template>
  </b-navbar>
</template>

<script>
  import ky from "ky";

  export default {
    data() {
      return {
        currentVersion: "",
        latestVersion: "",
      }
    },
    mounted() {
      let d = document.documentElement;
      d.className += " has-navbar-fixed-top";

      ky.get(`/api/config/version-check`).json().then(data => {
        this.currentVersion = data.current_version;
        this.latestVersion = data.latest_version;

        if (this.currentVersion !== this.latestVersion && this.currentVersion !== "CURRENT") {
          this.$buefy.snackbar.open({
            message: `Version ${this.latestVersion} available!`,
            type: 'is-warning',
            position: 'is-top',
            actionText: 'Download now',
            indefinite: true,
            onAction: () => {
              window.location = "https://github.com/cld9x/xbvr/releases";
            }
          })
        }
      });
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
</style>