<template>
  <div class="field">
    <section>
      <div class="columns">
        <div class="column is-one-fifth">
          <label class="label">State</label>
          <b-field>
            <b-radio-button v-model="fileState" native-value="all">
              <span>All</span>
            </b-radio-button>
            <b-radio-button v-model="fileState" native-value="matched">
              <span>Matched</span>
            </b-radio-button>
            <b-radio-button v-model="fileState" native-value="unmatched">
              Unmatched
            </b-radio-button>
          </b-field>
        </div>
        <div class="column is-one-fifth">
          <label class="label">Created between</label>
          <b-field>
            <b-datepicker v-model="fileCreation" editable range>
            </b-datepicker>
          </b-field>
        </div>
      </div>
    </section>
  </div>
</template>

<script>
  export default {
    name: "Filters",
    computed: {
      fileState: {
        get() {
          return this.$store.state.files.filters.state;
        },
        set(value) {
          this.$store.state.files.filters.state = value;
          this.$store.dispatch("files/load");
        }
      },
      fileCreation: {
        get() {
          return this.$store.state.files.filters.createdDate;
        },
        set(value) {
          console.log(value);
          this.$store.state.files.filters.createdDate = value;
          this.$store.dispatch("files/load");
        }
      },
    }
  }
</script>
