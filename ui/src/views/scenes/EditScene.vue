<template>
  <div class="modal is-active">
    <GlobalEvents
      @keyup.esc="close"
      @keyup.s="save" />

    <div class="modal-background"></div>

    <div class="modal-card">
      <header class="modal-card-head">
        <p class="modal-card-title">{{ $t('Edit Scene Details') }}</p>
        <button class="delete" @click="close" aria-label="close"></button>
      </header>

      <section class="modal-card-body">
        <b-field :label="$t('Title')">
          <b-input type="text" v-model="title" />
        </b-field>

        <b-field :label="$t('Description')">
          <b-input type="textarea" v-model="synopsis" />
        </b-field>

        <b-field grouped group-multiline>
          <b-field :label="$t('Studio')">
            <b-input type="text" v-model="studio" />
          </b-field>

          <b-field :label="$t('Site')">
            <b-input type="text" v-model="site" />
          </b-field>

          <b-field :label="$t('Scene URL')">
            <b-input type="text" v-model="scene_url" />
          </b-field>

          <b-field :label="$t('Release Date')">
            <div class="control">
              <input type="date" class="input" v-model="release_date_text" />
            </div>
          </b-field>
        </b-field>

        <b-field :label="$t('Cast')">
          <b-taginput type="is-warning" icon="label" v-model="castArray" />
        </b-field>

        <b-field :label="$t('Tags')">
          <b-taginput type="is-info" icon="label" v-model="tagsArray" />
        </b-field>

        <b-field>
          <b-button type="is-primary">{{ $t('Save Scene Details') }}</b-button>
        </b-field>
      </section>
    </div>
  </div>
</template>

<script>
  import GlobalEvents from 'vue-global-events';

  export default {
    name: "EditScene",
    components: {GlobalEvents},
    data() {
      /*
      title: string,
      synopsis: string,
      release_date_text: string,
      studio: string,
      site: string,
      scene_url: string,
      cast: object[]
      tags: object[]
       */
      const scene = this.$store.state.overlay.edit.scene;
      scene.castArray = scene.cast.map(c => c.name);
      scene.tagsArray = scene.tags.map(t => t.name);
      return scene;
    },
    computed: {
      item() {
        return this.$store.state.overlay.edit.scene;
      },
    },
    methods: {
      close() {
        this.$store.commit("overlay/hideEditDetails");
      },
      save() {

      }
    }
  }
</script>

<style scoped>
  .modal-card {
    width: 40%;
  }
</style>
