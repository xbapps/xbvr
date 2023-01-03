<template>
  <div class="container">
    <b-loading :is-full-page="false" :active.sync="isLoading"></b-loading>
    <b-tabs v-model="activeTab" size="medium" type="is-boxed" style="margin-left: 0px" id="playertab">
      <b-tab-item label="Shared Settings"/>
      <b-tab-item label="DeoVR"/>
      <b-tab-item label="Heresphere"/>
    </b-tabs>
    <div class="content" v-if="activeTab == 0">
      <h3>Shared Player Options</h3>
      <hr/>
      <div class="columns">
        <div class="column">
          <section>
            <b-field label="Player integration">
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
                  If you are using funscripts, you can add a heatmap to the thumbnails of scripted scenes in the Player interface.
                </p>
              </div>
              <hr/>
              <div class="block">
                <b-field label="Watch time tracking">
                  <b-switch v-model="watchTimeTrackingEnabled">
                    Enabled
                  </b-switch>
                </b-field>
              </div>
            </div>
          </section>
        </div>
        <div class="column content">
          <p>
            {{ $t("Player interface is available at following URLs:") }}
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
    </div>
    <div class="content" v-if="activeTab == 1">
      <h3>DeoVR interface</h3>
      <hr/>
      <div class="block">
          <b-field label="Remote mode">
            <b-switch v-model="remoteEnabled" :disabled="watchTimeTrackingEnabled === false">
              Enabled
            </b-switch>
          </b-field>
          <p>
            Requires: Watch time tracking
          </p>
          <p>
            To use remote mode, which enables more precise watch time tracking, you need to turn it on in DeoVR
            settings too - see <a href="https://deovr.com/doc#remote-control" target="_blank" rel="noreferrer">
            instructions in DeoVR documentation</a>.
          </p>
        </div>
    </div>
    <div class="content" v-if="activeTab == 2">
      <h3>Heresphere interface</h3>
      <hr/>
          <b-tooltip
            label="WANRING: file deletes from Heresphere are permanent. ALL files associated with a scene will be deleted"
            size="is-large" type="is-danger" multilined :delay="250" >
            <b-field label="Allow File Deletion">
              <b-switch v-model="allowFileDeletions">
                Enabled
              </b-switch>
            </b-field>
          </b-tooltip>
          <b-field label="Allow Ratings Updates">
            <b-switch v-model="allowRatingUpdates">
              Enabled
            </b-switch>
          </b-field>
          <b-field label="Allow Favorite Updates">
            <b-switch v-model="allowFavouriteUpdates">
              Enabled
            </b-switch>
          </b-field>
          <b-field label="Allow Tag Updates">
            <b-switch v-model="allowTagUpdates">
              Enabled
            </b-switch>
          </b-field>
          <b-field label="Allow Cuepoint Updates">
            <b-switch v-model="allowCuepointUpdates">
              Enabled
            </b-switch>
          </b-field>
          <b-tooltip
            label="Add or delete the Feature:watchlist tag to toggle the Watchlist flag in XBVR"
            size="is-large" type="is-primary" multilined :delay="250" >
            <b-field label="Allow Watchlist Updates">
              <b-switch v-model="allowWatchlistUpdates">
                Enabled
              </b-switch>
            </b-field>
          </b-tooltip>
          <b-field label="Allow Saving Hsp Files">
            <b-switch v-model="allowHspData">
              Enabled
            </b-switch>
          </b-field>
          <div class="columns">
            <div class="column is-one-half"> 
              <b-tooltip
                label="This option will split Cuepoints into multiple tracks, eg Standing-Doggy will split into 2 tracks in Heresphere"
                size="is-large" type="is-primary" multilined :delay="250" >
                <b-field label="Use Multi-Track Cuepoints">
                  <b-switch v-model="multiTrackCuepoints">
                    Enabled
                  </b-switch>
                </b-field>
              </b-tooltip>
            </div>
            <div class="column is-one-half"> 
              <b-tooltip
                label="This option will split Cuepoints matching the Actors Name into seperate tracks in Heresphere"
                size="is-large" type="is-primary" multilined :delay="250" >
                <b-field label="Use Multi-Track Cast Cuepoints">
                  <b-switch v-model="multiTrackCastCuepoints">
                    Enabled
                  </b-switch>
                </b-field>
              </b-tooltip>
            </div>
          </div>
      </div>
      <b-field>
        <b-button type="is-primary" @click="save">Save and apply changes</b-button>
      </b-field>
    </div>
</template>

<script>
export default {
  name: 'InterfaceDeoVR',
  mounted () {
    this.$store.dispatch('optionsDeoVR/load')
  },
  data () {
    return {
      activeTab: 0
    }
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
    watchTimeTrackingEnabled: {
      get () {
        return this.$store.state.optionsDeoVR.deovr.track_watch_time
      },
      set (value) {
        this.$store.state.optionsDeoVR.deovr.track_watch_time = value
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
    allowFileDeletions: {
      get () {
        return this.$store.state.optionsDeoVR.heresphere.allow_file_deletes
      },
      set (value) {
        this.$store.state.optionsDeoVR.heresphere.allow_file_deletes= value
      }
    },
    allowRatingUpdates: {
      get () {
        return this.$store.state.optionsDeoVR.heresphere.allow_rating_updates
      },
      set (value) {
        this.$store.state.optionsDeoVR.heresphere.allow_rating_updates = value
      }
    },
    allowFavouriteUpdates: {
      get () {
        return this.$store.state.optionsDeoVR.heresphere.allow_favorite_updates
      },
      set (value) {
        this.$store.state.optionsDeoVR.heresphere.allow_favorite_updates = value
      }
    },
    allowTagUpdates: {
      get () {
        return this.$store.state.optionsDeoVR.heresphere.allow_tag_updates
      },
      set (value) {
        this.$store.state.optionsDeoVR.heresphere.allow_tag_updates = value
      }
    },
    allowCuepointUpdates: {
      get () {
        return this.$store.state.optionsDeoVR.heresphere.allow_cuepoint_updates
      },
      set (value) {
        this.$store.state.optionsDeoVR.heresphere.allow_cuepoint_updates = value
      }
    },
    allowWatchlistUpdates: {
      get () {
        return this.$store.state.optionsDeoVR.heresphere.allow_watchlist_updates
      },
      set (value) {
        this.$store.state.optionsDeoVR.heresphere.allow_watchlist_updates = value
      }
    },
    allowHspData: {
      get () {
        return this.$store.state.optionsDeoVR.heresphere.allow_hsp_data
      },
      set (value) {
        this.$store.state.optionsDeoVR.heresphere.allow_hsp_data= value
      }
    },    
    multiTrackCuepoints: {
      get () {
        return this.$store.state.optionsDeoVR.heresphere.multitrack_cuepoints
      },
      set (value) {
        this.$store.state.optionsDeoVR.heresphere.multitrack_cuepoints = value
      }
    },
    multiTrackCastCuepoints: {
      get () {        
        return this.$store.state.optionsDeoVR.heresphere.multitrack_cast_cuepoints
      },
      set (value) {
        this.$store.state.optionsDeoVR.heresphere.multitrack_cast_cuepoints = value
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
