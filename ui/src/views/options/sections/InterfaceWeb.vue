<template>
  <div class="container">
    <b-loading :is-full-page="false" :active.sync="isLoading" />
    <div class="content">
      <h3>{{ $t('Web UI') }}</h3>
      <hr />
      <div class="columns">
        <div class="column">
          <section>
            <b-field label="Tag Sort">
              <div class="block">
                <b-radio v-model="tagSort" name="tagSort" native-value="by-tag-count">
                  By Tag Count
                </b-radio>
                <b-radio v-model="tagSort" name="tagSort" native-value="alphabetically">
                  Alphabetically
                </b-radio>
              </div>
            </b-field>

            <b-field label="Scene Edit in List">
              <b-switch v-model="sceneEdit">
                Enabled
              </b-switch>
            </b-field>

            <b-field label="Automatically Check for Updates">
              <b-switch v-model="updateCheck">
                Enabled
              </b-switch>
            </b-field>

            <b-field>
              <b-button type="is-primary" @click="save">Save</b-button>
            </b-field>
          </section>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  name: 'InterfaceWeb',
  mounted () {
    this.$store.dispatch('optionsWeb/load')
  },
  methods: {
    save () {
      this.$store.dispatch('optionsWeb/save')
    }
  },
  computed: {
    tagSort: {
      get () {
        return this.$store.state.optionsWeb.web.tagSort
      },
      set (value) {
        this.$store.state.optionsWeb.web.tagSort = value
      }
    },
    sceneEdit: {
      get () {
        return this.$store.state.optionsWeb.web.sceneEdit
      },
      set (value) {
        this.$store.state.optionsWeb.web.sceneEdit = value
      }
    },
    updateCheck: {
      get () {
        return this.$store.state.optionsWeb.web.updateCheck
      },
      set (value) {
        this.$store.state.optionsWeb.web.updateCheck = value
      }
    },
    isLoading: function () {
      return this.$store.state.optionsWeb.loading
    }
  }
}
</script>

<style scoped>

</style>
