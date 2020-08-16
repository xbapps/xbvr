<template>
  <div class="modal is-active">
    <GlobalEvents
      :filter="e => !['INPUT', 'TEXTAREA'].includes(e.target.tagName)"
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
          <b-input type="text" v-model="scene.title" />
        </b-field>

        <b-field :label="$t('Description')">
          <b-input type="textarea" v-model="scene.synopsis" />
        </b-field>

        <b-field grouped group-multiline>
          <b-field :label="$t('Studio')">
            <b-input type="text" v-model="scene.studio" />
          </b-field>

          <b-field :label="$t('Site')">
            <b-input type="text" v-model="scene.site" />
          </b-field>

          <b-field :label="$t('Scene URL')">
            <b-input type="text" v-model="scene.scene_url" />
          </b-field>

          <b-field :label="$t('Release Date')">
            <div class="control">
              <input type="date" class="input" v-model="scene.release_date_text" />
            </div>
          </b-field>
        </b-field>

        <b-field :label="$t('Cast')">
          <b-taginput type="is-warning"
                      icon="label"
                      placeholder="Add an actor"
                      v-model="scene.castArray"
                      autocomplete
                      :allow-new="true"
                      :allow-duplicates="false"
                      :data="filteredCast"
                      @typing="getFilteredCast" />
        </b-field>

        <b-field :label="$t('Tags')">
          <b-taginput type="is-info"
                      icon="label"
                      placeholder="Add a tag"
                      v-model="scene.tagsArray"
                      autocomplete
                      :allow-new="true"
                      :allow-duplicates="false"
                      :data="filteredTags"
                      @typing="getFilteredTags" />
        </b-field>

        <b-field grouped>
          <b-button class="control" type="is-primary" outlined @click="editFilenames">Edit filenames</b-button>
          <b-button class="control" type="is-primary" outlined @click="editCovers">Edit covers</b-button>
          <b-button class="control" type="is-primary" outlined @click="editGallery">Edit gallery</b-button>
        </b-field>

        <b-field>
          <b-button type="is-primary" @click="save">{{ $t('Save Scene Details') }}</b-button>
        </b-field>
      </section>

      <ListEditor v-if="showListEditor" />
    </div>
  </div>
</template>

<script>
  import ky from "ky";
  import GlobalEvents from 'vue-global-events';
  import ListEditor from "../../components/ListEditor";

  export default {
    name: "EditScene",
    components: {ListEditor, GlobalEvents},
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
      images: object[]
      filenames_arr: string[]
       */
      const scene = Object.assign({}, this.$store.state.overlay.edit.scene);
      scene.castArray = scene.cast.map(c => c.name);
      scene.tagsArray = scene.tags.map(t => t.name);
      const images = JSON.parse(scene.images);
      scene.covers = images.filter(i => i.type === 'cover').map(i => i.url);
      scene.gallery = images.filter(i => i.type === 'gallery').map(i => i.url);
      scene.files = JSON.parse(scene.filenames_arr);
      return {
        scene,
        filteredCast: [],
        filteredTags: [],
      };
    },
    methods: {
      getFilteredCast(text) {
        this.filteredCast = this.filters.cast.filter(option =>
          option.toString().toLowerCase().indexOf(text.toLowerCase()) >= 0);
      },
      getFilteredTags(text) {
        this.filteredTags = this.filters.tags.filter(option =>
          option.toString().toLowerCase().indexOf(text.toLowerCase()) >= 0);
      },
      close() {
        this.$store.commit("overlay/hideEditDetails");
      },
      save() {
        const images = [];
        this.scene.covers.forEach(url => {
          images.push({
            url,
            type: "cover",
            orientation: "",
          });
        });
        this.scene.gallery.forEach(url => {
          images.push({
            url,
            type: "gallery",
            orientation: "",
          })
        });
        this.scene.images = JSON.stringify(images);
        this.scene.cover_url = this.scene.covers[0];
        this.scene.filenames_arr = JSON.stringify(this.scene.files);

        ky.post(`/api/scene/edit/${this.scene.id}`, {json: {...this.scene}});

        this.scene.cast = this.scene.castArray.map(a => {
          const find = this.scene.cast.find(o => o.name === a);
          if (find) return find;
          return {
            name: a,
            count: 0,
          }
        });

        this.scene.tags = this.scene.tagsArray.map(t => {
          const find = this.scene.tags.find(o => o.name === t);
          if (find) return find;
          return {
            name: t,
            count: 0,
          }
        })

        this.$store.commit('sceneList/updateScene', this.scene);

        this.close();
      },
      editFilenames() {
        this.$store.commit('overlay/showListEditor', {
          list: this.scene.files,
          label: "Edit Scene Filenames",
        });
      },
      editCovers() {
        this.$store.commit('overlay/showListEditor', {
          list: this.scene.covers,
          label: "Edit Scene Covers",
        });
      },
      editGallery() {
        this.$store.commit('overlay/showListEditor', {
          list: this.scene.gallery,
          label: "Edit Scene Gallery",
        });
      },
    },
    computed: {
      filters() {
        return this.$store.state.sceneList.filterOpts;
      },
      showListEditor() {
        return this.$store.state.overlay.listEditor.show;
      }
    }
  }
</script>

<style scoped>
  .modal-card {
    width: 40%;
  }
</style>
