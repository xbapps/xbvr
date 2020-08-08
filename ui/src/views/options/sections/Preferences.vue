<template>
  <div class="container">
    <b-loading :is-full-page="false" :active.sync="isLoading" />
    <div class="content">
      <h3>{{ $t('Preferences') }}</h3>
      <hr />
      <div class="columns">
        <div class="column">
          <section>
            <b-field label="Tag Sort">
              <b-switch v-model="tagSort" true-value="Alphabetically" false-value="By Tag Count">
                {{ tagSort }}
              </b-switch>
            </b-field>

            <b-field>
              <b-button type="is-primary" @click="save">Save Preferences</b-button>
            </b-field>
          </section>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
  export default {
    name: 'Preferences',
    mounted() {
      this.$store.dispatch("preferences/load");
    },
    methods: {
      save() {
        this.$store.dispatch("preferences/save");
      },
    },
    computed: {
      tagSort: {
        get() {
          return this.$store.state.preferences.prefs.tagSort;
        },
        set(value) {
          this.$store.state.preferences.prefs.tagSort = value;
        }
      },
      isLoading: function() {
        return this.$store.state.preferences.loading;
      }
    }
  }
</script>

<style scoped>

</style>
