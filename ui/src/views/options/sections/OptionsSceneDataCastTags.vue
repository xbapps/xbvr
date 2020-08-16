<template>
  <div class="content">
    <div class="columns">
      <div class="column">
        <section>
          <b-field :label="$t('Cast')">
            <a class="button is-primary" v-on:click="cleanNames">{{$t('Clean model names')}}</a>
          </b-field>

          <b-field :label="$t('Tags')">
            <a class="button is-primary" v-on:click="cleanTags">{{$t('Clean tags')}}</a>
          </b-field>
        </section>
      </div>
    </div>
  </div>
</template>

<script>
  import ky from "ky";

  export default {
    name: "OptionsSceneDataCastTags",
    methods: {
      cleanNames() {
        ky.get(`/api/task/clean-cast`);
      },
      cleanTags() {
        ky.get(`/api/task/clean-tags`);
      }
    }
  }
</script>

<style scoped>

</style>
