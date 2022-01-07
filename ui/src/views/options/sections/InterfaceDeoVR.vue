<template>
  <div class="container">
    <b-loading :is-full-page="false" :active.sync="isLoading"></b-loading>
    <div class="content">
      <h3>{{ $t("DeoVR interface") }}</h3>
      <hr/>
      <div class="columns">
        <div class="column">
          <section>
            <b-field label="DeoVR integration">
              <b-switch v-model="enabled">
                Enabled
              </b-switch>
            </b-field>
            <hr/>
            <div v-if="enabled">
              <b-field label="Authentication">
                <b-switch v-model="authEnabled">
                  Enabled
                </b-switch>
              </b-field>
              <div class="block">
                <b-field grouped>
                  <b-field label="Username">
                    <b-input v-model="username" :disabled="authEnabled === false" style="width:200px"></b-input>
                  </b-field>
                  <b-field label="Password">
                    <b-input v-model="password" :disabled="authEnabled === false" style="width:200px" type="password"></b-input>
                  </b-field>
                </b-field>
              </div>
              <hr/>
              <div class="block">
                <b-field label="Funscript heatmaps">
                  <b-switch v-model="renderHeatmaps">
                    Enabled
                  </b-switch>
                </b-field>
                <p>
                  If you are using funscripts, you can add a heatmap to the thumbnails of scripted scenes in the DeoVR interface.
                </p>
              </div>
              <hr/>
              <div class="block">
                <b-field label="Remote mode">
                  <b-switch v-model="remoteEnabled">
                    Enabled
                  </b-switch>
                </b-field>
                <p>
                  To use remote mode, which enables more precise watch time tracking, you need to turn it on in DeoVR
                  settings too - see <a href="https://deovr.com/doc#remote-control" target="_blank" rel="noreferrer">
                  instructions in DeoVR documentation</a>.
                </p>
              </div>
            </div>
          </section>
        </div>
        <div class="column content">
          <p>
            {{ $t("DeoVR interface is available at following URLs:") }}
          </p>
          <div>
            <h4 v-for="(addr, idx) in boundIp" :key="'ip' + idx">{{ addr }}</h4>
          </div>
          <hr/>
          <p>
            NOTE: make sure DeoVR is using <strong>http://</strong> not <strong>https://</strong>.<br/>
            To toggle used protocol, click on it in DeoVR's URL bar.
          </p>
        </div>
      </div>

      <b-field>
        <b-button type="is-primary" @click="save">Save and apply changes</b-button>
      </b-field>
    </div>
  </div>
</template>

<script>
export default {
  name: 'InterfaceDeoVR',
  mounted () {
    this.$store.dispatch('optionsDeoVR/load')
  },
  methods: {
    save () {
      this.$store.dispatch('optionsDeoVR/save')
    },
    addIP (value) {
      const tmp = [...this.allowedIp]
      tmp.push(value)

      if (!this.hasDuplicates(tmp)) {
        this.allowedIp = tmp
      }
    },
    hasDuplicates (array) {
      return (new Set(array)).size !== array.length
    }
  },
  computed: {
    enabled: {
      get () {
        return this.$store.state.optionsDeoVR.deovr.enabled
      },
      set (value) {
        this.$store.state.optionsDeoVR.deovr.enabled = value
      }
    },
    authEnabled: {
      get () {
        return this.$store.state.optionsDeoVR.deovr.auth_enabled
      },
      set (value) {
        this.$store.state.optionsDeoVR.deovr.auth_enabled = value
      }
    },
    renderHeatmaps: {
      get () {
        return this.$store.state.optionsDeoVR.deovr.render_heatmaps
      },
      set (value) {
        this.$store.state.optionsDeoVR.deovr.render_heatmaps = value
      }
    },
    remoteEnabled: {
      get () {
        return this.$store.state.optionsDeoVR.deovr.remote_enabled
      },
      set (value) {
        this.$store.state.optionsDeoVR.deovr.remote_enabled = value
      }
    },
    username: {
      get () {
        return this.$store.state.optionsDeoVR.deovr.username
      },
      set (value) {
        this.$store.state.optionsDeoVR.deovr.username = value
      }
    },
    password: {
      get () {
        return this.$store.state.optionsDeoVR.deovr.password
      },
      set (value) {
        this.$store.state.optionsDeoVR.deovr.password = value
      }
    },
    boundIp: {
      get () {
        return this.$store.state.optionsDeoVR.deovr.boundIp
      }
    },
    isLoading: function () {
      return this.$store.state.optionsDeoVR.loading
    },
    deoVROptions: function () {
      return this.$store.state.optionsDeoVR.deovr
    }
  }
}
</script>
