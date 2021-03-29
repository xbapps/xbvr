<template>
  <div class="modal is-active">
    <GlobalEvents
      :filter="e => !['INPUT', 'TEXTAREA'].includes(e.target.tagName)"
      @keyup.esc="close"
      @keyup.s="save"/>

    <div class="modal-background"></div>

    <div class="modal-card">
      <header class="modal-card-head">
        <p class="modal-card-title">{{ $t('Edit scene details') }}</p>
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

          <b-tab-item :label="$t('Covers')">
            <ListEditor :list="this.scene.covers" type="covers" :blurFn="() => blur('covers')"/>
          </b-tab-item>

          <b-tab-item :label="$t('Gallery')">
            <ListEditor :list="this.scene.gallery" type="gallery" :blurFn="() => blur('gallery')"/>
          </b-tab-item>
        </b-tabs>

      </section>

      <footer class="modal-card-foot">
        <b-field>
          <b-button type="is-primary" @click="save">{{ $t('Save Scene Details') }}</b-button>
        </b-field>
      </footer>
    </div>
  </div>
</template>

<script>
import ky from 'ky'
import GlobalEvents from 'vue-global-events'
import ListEditor from '../../components/ListEditor'

export default {
  name: 'EditScene',
  components: { ListEditor, GlobalEvents },
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
    const images = JSON.parse(scene.images)
    scene.covers = images.filter(i => i.type === 'cover').map(i => i.url)
    scene.gallery = images.filter(i => i.type === 'gallery').map(i => i.url)
    scene.files = JSON.parse(scene.filenames_arr)
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
      const images = []
      this.scene.covers.forEach(url => {
        images.push({
          url,
          type: 'cover',
          orientation: ''
        })
      })
      this.scene.gallery.forEach(url => {
        images.push({
          url,
          type: 'gallery',
          orientation: ''
        })
      })
      this.scene.images = JSON.stringify(images)
      this.scene.cover_url = this.scene.covers[0]
      this.scene.filenames_arr = JSON.stringify(this.scene.files)

      ky.post(`/api/scene/edit/${this.scene.id}`, { json: { ...this.scene } })

      this.scene.cast = this.scene.castArray.map(a => {
        const find = this.scene.cast.find(o => o.name === a)
        if (find) return find
        return {
          name: a,
          count: 0
        }
      })

      this.scene.tags = this.scene.tagsArray.map(t => {
        const find = this.scene.tags.find(o => o.name === t)
        if (find) return find
        return {
          name: t,
          count: 0
        }
      })

      this.$store.commit('sceneList/updateScene', this.scene)
      this.changesMade = false

      this.close()
    },
    blur (field) {
      if (this.changesMade) return // Changes have already been made. No point to check any further
      if (['castArray', 'tagsArray', 'files', 'covers', 'gallery'].includes(field)) {
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
