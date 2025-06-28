<template>
  <div class="modal is-active">
    <GlobalEvents
      :filter="e => !['INPUT', 'TEXTAREA'].includes(e.target.tagName)"
      @keyup.esc="close"
      @keyup.s="save"/>

    <div class="modal-background"></div>

    <div class="modal-card">
      <header class="modal-card-head">
        <p class="modal-card-title">{{ this.scene.id == 0 ? $t('Display scene details') : $t('Edit scene details') }}</p>
        <button class="delete" @click="close" aria-label="close"></button>
      </header>

      <section class="modal-card-body">
        <b-tabs position="is-centered" :animated="false">

          <b-tab-item :label="$t('Information')">
            <b-field :label="$t('Title')">
              <b-input type="text" v-model="scene.title" @blur="blur('title')"/>
            </b-field>

            <b-field :label="$t('Multipart scene')">
              <b-checkbox v-model="scene.is_multipart"/>
            </b-field>

            <b-field grouped group-multiline>
              <b-field :label="$t('Studio')">
                <b-input type="text" v-model="scene.studio" @blur="blur('studio')"/>
              </b-field>

              <b-field :label="$t('Site')">
                <b-input type="text" v-model="scene.site" @blur="blur('site')"/>
              </b-field>

              <b-field :label="$t('Scene URL')">
                <b-input type="text" v-model="scene.scene_url" @blur="blur('scene_url')"/>
              </b-field>

              <b-field :label="$t('Release Date')">
                <div class="control">
                  <input type="date" class="input" v-model="scene.release_date_text"
                         @blur="blur('release_date_text')"/>
                </div>
              </b-field>

              <b-field :label="$t('Duration')">
                <b-input type="number" v-model="scene.duration" @blur="blur('duration')"/>
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
                          @typing="getFilteredCast"
                          @blur="blur('castArray')"/>
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
                          @typing="getFilteredTags"
                          @blur="blur('tagsArray')"/>
            </b-field>

            <b-field :label="$t('Description')">
              <b-input type="textarea" v-model="scene.synopsis" @blur="blur('synopsis')"/>
            </b-field>
          </b-tab-item>

          <b-tab-item :label="$t('Filenames')">
            <ListEditor :list="this.scene.files" type="files" :blurFn="() => blur('files')"/>
          </b-tab-item>

          <b-tab-item :label="$t('Gallery')">
            <GalleryEditor
              :list.sync="scene.gallery"
              :blurFn="() => blur('gallery')"
              :coverUrl="this.scene.cover_url"
              @setCover="setCoverImage"
            />
          </b-tab-item>
        </b-tabs>

      </section>

      <footer class="modal-card-foot is-justify-content-space-between">
        <b-button type="is-primary" @click="save">{{ $t('Save Scene Details') }}</b-button>
        <b-button v-if="this.scene.id != 0" type="is-danger" outlined @click="deletescene">{{ $t('Delete Scene') }}</b-button>
      </footer>
    </div>
  </div>
</template>

<script>
import ky from 'ky'
import GlobalEvents from 'vue-global-events'
import ListEditor from '../../components/ListEditor'
import GalleryEditor from '../../components/GalleryEditor'

export default {
  name: 'EditScene',
  components: { ListEditor, GlobalEvents, GalleryEditor },
  data () {
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
    is_multipart: bool
     */
    const scene = Object.assign({}, this.$store.state.overlay.edit.scene)
    scene.castArray = scene.cast.map(c => c.name)
    scene.tagsArray = scene.tags.map(t => t.name)
    let images
    try {
      images = JSON.parse(scene.images)
    } catch {
      images = []
    }

    try {
      // map all scene images into the gallery list
      scene.gallery = images.map(i => i.url)
      // -If- there is -no- explicit cover_url, set the first image as cover
      if (!scene.cover_url && scene.gallery.length > 0) {
        scene.cover_url = scene.gallery[0]
      }
    } catch { 
      scene.gallery = []
    }

    try {
      // Fetch the image filenames
      scene.files = JSON.parse(scene.filenames_arr)
      if (scene.files == null) {
        scene.files = []
      }
    } catch {
      scene.files = []
    }
    return {
      scene,
      // A shallow copy won't work, need a deep copy
      source: JSON.parse(JSON.stringify(scene)),
      filteredCast: [],
      filteredTags: [],
      changesMade: false
    }
  },
  methods: {
    getFilteredCast (text) {
      this.filteredCast = this.filters.cast.filter(option => (
        option.toString().toLowerCase().indexOf(text.toLowerCase()) >= 0) &&
        !this.scene.cast.some(entry => entry.name === option.toString())
      )
    },
    getFilteredTags (text) {
      this.filteredTags = this.filters.tags.filter(option => (
        option.toString().toLowerCase().indexOf(text.toLowerCase()) >= 0) &&
        !this.scene.tags.some(entry => entry.name === option.toString())
      )
    },
    close () {
      if (this.changesMade) {
        this.$buefy.dialog.confirm({
          title: 'Close without saving',
          message: 'Are you sure you want to close before saving your changes?',
          confirmText: 'Close',
          type: 'is-warning',
          hasIcon: true,
          onConfirm: () => this.$store.commit('overlay/hideEditDetails')
        })
        return
      }
      this.$store.commit('overlay/hideEditDetails')
    },
    save () {
      // If there are images in the gallery, ensure a cover is set and gallery is normalized
      if (this.scene.gallery.length > 0) {
        // If no cover is set, use the first image as cover
        let coverUrl = this.scene.cover_url || this.scene.gallery[0];
        this.scene.cover_url = coverUrl;
        // Normalize gallery: cover first, no duplicates
        this.scene.gallery = [coverUrl, ...this.scene.gallery.filter(url => url !== coverUrl)];
      }

      // Load original image metadata (from DB) to preserve type and orientation
      let originalImages = [];
      try {
        originalImages = JSON.parse(this.source.images);
      } catch {
        originalImages = [];
      }

      // Build images metadata array
      const seen = new Set();
      const images = this.scene.gallery.reduce((arr, url) => {
        if (seen.has(url)) return arr;
        seen.add(url);
        const existing = originalImages.find(img => img.url === url);
        let type = existing?.type;
        const orientation = existing?.orientation || '';
        if (type !== 'cover' && type !== 'gallery') {
          type = (url === this.scene.cover_url) ? 'cover' : 'gallery';
        }
        arr.push({ url, type, orientation });
        return arr;
      }, []);
      this.scene.images = JSON.stringify(images);

      this.scene.filenames_arr = JSON.stringify(this.scene.files);
      this.scene.duration = String(this.scene.duration);

      // Push to backend with proper error handling
      ky.post(`/api/scene/edit/${this.scene.id}`, { json: { ...this.scene } })
        .json()
        .then(data => {
          this.$store.commit('sceneList/updateScene', data);
          this.$store.commit('overlay/showDetails', { scene: data });
          // Reset changesMade flag after successful save
          this.changesMade = false;
          this.close();
        })
        // On error, don't reset changesMade flag or close modal.User can try saving again
        .catch(error => {
          console.error('Failed to save scene:', error);
          // Show user-friendly error message
          this.$buefy.toast.open({
            message: 'Failed to save scene changes. Please try again.',
            type: 'is-danger',
            duration: 5000
          });
        });
    },
    deletescene () {
      this.$buefy.dialog.confirm({
        title: 'Delete scene',
        message: `Do you really want to delete the scene <strong>${this.scene.title}</strong> from <strong>${this.scene.studio}</strong>? If this is an existing scene, it will be re-added during the next scrape.`,
        type: 'is-info is-wide',
        hasIcon: true,
        id: 'heh',
        onConfirm: () => {
          ky.post(`/api/scene/delete`, {json:{scene_id: this.scene.id}}).json().then(data => {
            this.$store.dispatch('sceneList/load', { offset: 0 })
            this.$store.commit('overlay/hideEditDetails')
            this.$store.commit('overlay/hideDetails')
          })
        }
      })
    },
    blur (field) {
      if (this.changesMade) return // Changes have already been made. No point to check any further
      
      // Removed 'covers'as this is now handled in the GalleryEditor component
      if (['castArray', 'tagsArray', 'files', 'gallery'].includes(field)) {
        if (this.scene[field].length !== this.source[field].length) {
          this.changesMade = true
        } else {
          for (let i = 0; i < this.scene[field].length; i++) {
            if (this.scene[field][i] !== this.source[field][i]) {
              this.changesMade = true
              break
            }
          }
        }
      } else if (this.scene[field] !== this.source[field]) {
        this.changesMade = true
      }
    },
    // Update displayed cover image in the UI
    setCoverImage (url) {
      this.scene.cover_url = url
      this.changesMade = true
    }
  },
  computed: {
    filters () {
      return this.$store.state.sceneList.filterOpts
    }
  }
}
</script>

<style scoped>
.modal-card {
  width: 65%;
}

.tab-item {
  height: 40vh;
}
</style>
