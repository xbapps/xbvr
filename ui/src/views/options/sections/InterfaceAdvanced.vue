<template>
  <div class="container">
    <b-loading :is-full-page="false" :active.sync="isLoading" />
    <div class="content">
      <h3>{{ $t('Advanced') }}</h3>
      <hr />
      <div class="columns">
        <div class="column">
          <section>
            <b-field>
              <b-switch v-model="showInternalSceneId" type="is-default">
                show Internal Scene Id
              </b-switch>
            </b-field>
            <b-field>
              <b-switch v-model="showHSPApiLink" type="is-default">
                show Heresphere Api Link
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
  name: 'InterfaceAdvanced',
  mounted () {    
    this.$store.dispatch('optionsAdvanced/load')    
  },
  methods: {
    save () {
      this.$store.dispatch('optionsAdvanced/save')
    }
  },
  computed: {
    showInternalSceneId: {
      get () {
        return this.$store.state.optionsAdvanced.advanced.showInternalSceneId
      },
      set (value) {
        this.$store.state.optionsAdvanced.advanced.showInternalSceneId = value
      }
    },
    showHSPApiLink: {
      get () {
        return this.$store.state.optionsAdvanced.advanced.showHSPApiLink
      },
      set (value) {
        this.$store.state.optionsAdvanced.advanced.showHSPApiLink = value
      }
    },    
    isLoading: function () {
      return this.$store.state.optionsAdvanced.loading
    }
  }
}
</script>

<style scoped>

</style>
