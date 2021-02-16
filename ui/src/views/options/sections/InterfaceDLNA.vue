<template>
  <div class="container">
    <b-loading :is-full-page="false" :active.sync="isLoading"></b-loading>
    <div class="content">
      <h3>{{$t("DLNA interface")}}</h3>
      <hr/>
      <div class="columns">
        <div class="column">
          <section>
            <b-field label="DLNA server">
              <b-switch v-model="enabled">
                Enabled
              </b-switch>
            </b-field>

            <b-field label="Visible name">
              <b-input v-model="name" style="width:200px"></b-input>
            </b-field>

            <b-field grouped>
              <b-field label="Icon">
                <b-select placeholder="Select image" v-model="image">
                  <option v-for="s in dlnaOptions.availableImages" :value="s" :key="s.id">
                    {{ s }}
                  </option>
                </b-select>
              </b-field>
              <b-field label=" ">
                <img :src="`/ui/dlna/${image}.png`" width="64" style="margin-left:2em" v-if="image"/>
              </b-field>
            </b-field>

            <b-field label="Allowed IP addresses">
              <b-taginput v-model="allowedIp" :allow-new="true" placeholder="Type in a IP address" class="is-half"></b-taginput>
            </b-field>

            <b-field>
              <p v-if="!isLoading">
                Recent IP addresses:
                <span v-if="dlnaOptions.recentIp.length > 0">
                  <b-tag rounded v-for="s in dlnaOptions.recentIp" :value="s" :key="s.id" style="text-decoration: underline; margin-right:0.25em; cursor: pointer;" type="is-info"><span @click="addIP(s)">{{ s }}</span></b-tag>
                </span>
                <span v-else>none (connect to DLNA at least once to find out device's IP address)</span>
              </p>
            </b-field>

            <b-field>
              <b-button type="is-primary" @click="save">Save and apply changes</b-button>
            </b-field>

          </section>
        </div>
        <div class="column content">
          <p>
            {{$t("Standard protocol that works with players such as: Skybox, Pigasus, Mobile Station VR, and others.")}}
          </p>
          <p>
            {{$t("Since it is broadcasted accross the whole local network, you might want to restrict access to selected IP addresses or disable it completely.")}}
          </p>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  name: 'InterfaceDLNA',
  mounted () {
    this.$store.dispatch('optionsDLNA/load')
  },
  methods: {
    save () {
      this.$store.dispatch('optionsDLNA/save')
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
        return this.$store.state.optionsDLNA.dlna.enabled
      },
      set (value) {
        this.$store.state.optionsDLNA.dlna.enabled = value
      }
    },
    name: {
      get () {
        return this.$store.state.optionsDLNA.dlna.name
      },
      set (value) {
        this.$store.state.optionsDLNA.dlna.name = value
      }
    },
    image: {
      get () {
        return this.$store.state.optionsDLNA.dlna.image
      },
      set (value) {
        this.$store.state.optionsDLNA.dlna.image = value
      }
    },
    allowedIp: {
      get () {
        return this.$store.state.optionsDLNA.dlna.allowedIp
      },
      set (value) {
        this.$store.state.optionsDLNA.dlna.allowedIp = value
      }
    },
    isLoading: function () {
      return this.$store.state.optionsDLNA.loading
    },
    dlnaOptions: function () {
      return this.$store.state.optionsDLNA.dlna
    }
  }
}
</script>
