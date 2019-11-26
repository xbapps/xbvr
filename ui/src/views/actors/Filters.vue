<template>
  <div>
    <div class="field">
      <label class="label">{{$t("Scenes")}}</label>
      <div class="control is-expanded">
        <div class="select is-fullwidth">
          <select v-model="scenesState">
            <option value="any">{{$t("Any")}}</option>
            <option value="available">{{$t("Available right now")}}</option>
          </select>
        </div>
      </div>
    </div>

    <div class="field">
      <label class="label">{{$t("Age")}}</label>
      <b-field>
          <b-slider v-model="ageRange" :min="18" :max="50" :step="1" lazy>
            <b-slider-tick :value="ageRange[0]" :key="ageRange[0]">{{ ageRange[0] }}</b-slider-tick>
            <b-slider-tick :value="ageRange[1]" :key="ageRange[1]">{{ ageRange[1] }}</b-slider-tick>
          </b-slider>
          </b-slider lazy>
      </b-field>
    </div>

    <div class="field">
      <label class="label">{{$t("Hair Color")}}</label>
      <div class="control is-expanded">
        <div class="select is-fullwidth">
          <select v-model="filteredHair">
            <option value="brown">{{$t("Brown")}}</option>
            <option value="black">{{$t("Black")}}</option>
          </select>
        </div>
      </div>
    </div>

    <div class="field">
      <label class="label">{{$t("Nationality")}}</label>
      <div class="control is-expanded">
        <div class="select is-fullwidth">
          <select v-model="filteredNationality">
            <option value="brown">{{$t("Brown")}}</option>
            <option value="black">{{$t("Black")}}</option>
          </select>
        </div>
      </div>
    </div>

    <div class="field">
      <label class="label">{{$t("Eye Color")}}</label>
      <div class="control is-expanded">
        <div class="select is-fullwidth">
          <select v-model="filteredEyes">
            <option value="brown">{{$t("Brown")}}</option>
            <option value="black">{{$t("Black")}}</option>
          </select>
        </div>
      </div>
    </div>

  </div>
</template>

<script>
  export default {
    name: "Filters",
    mounted() {
      this.$store.dispatch("actorList/filters");
    },
    data() {
      return {
        ageRange: [20, 32],
        filteredEyes: [],
        filteredHair: [],
        filteredNationality: [],
      }
    },
  }
</script>
