<template>
  <div class="field">
    <section>
      <div class="columns">
        <div class="column is-one-fifth">
          <label class="label">{{$t("State")}}</label>
          <b-field>
            <b-radio-button v-model="fileState" native-value="all">
              <span>{{$t("All")}}</span>
            </b-radio-button>
            <b-radio-button v-model="fileState" native-value="matched">
              <span>{{$t("Matched")}}</span>
            </b-radio-button>
            <b-radio-button v-model="fileState" native-value="unmatched">
              <span>{{$t("Unmatched")}}</span>
            </b-radio-button>
          </b-field>
        </div>
        <div class="column is-one-fifth">
          <label class="label">{{$t("Created between")}}</label>
          <b-field>
            <b-datepicker v-model="fileCreation" editable range>
              <div class="buttons">
                <b-button size="is-small" @click="setRange(subDays(new Date(), 7), new Date())">
                  <span>Last 7 days</span>
                </b-button>
                <b-button size="is-small" @click="setRange(subDays(new Date(), 14), new Date())">
                  <span>Last 14 days</span>
                </b-button>
                <b-button size="is-small" @click="setRange(subDays(new Date(), 30), new Date())">
                  <span>Last 30 days</span>
                </b-button>
              </div>
            </b-datepicker>
            <button class="button is-light" @click="clearRange">
              <b-icon pack="fas" icon="times" size="is-small"></b-icon>
            </button>
          </b-field>
        </div>
      </div>
    </section>
  </div>
</template>

<script>
  import {subDays} from "date-fns";

  export default {
    name: "Filters",
    methods: {
      clearRange() {
        this.fileCreation = [];
      },
      setRange(start, end) {
        this.fileCreation = [start, end];
      },
      subDays,
    },
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
          this.$store.state.files.filters.createdDate = value;
          this.$store.dispatch("files/load");
        }
      },
    }
  }
</script>
