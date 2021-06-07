<template>
  <b-modal :active.sync="isActive"
           :destroy-on-hide="false"
           has-modal-card
           trap-focus
           aria-role="dialog"
           aria-modal
           can-cancel>
    <b-field style="width:600px">
      <b-autocomplete
        ref="autocompleteInput"
        :data="data"
        placeholder="Find scene..."
        field="query"
        :loading="isFetching"
        @typing="getAsyncData"
        @select="option => showSceneDetails(option)"
        custom-class="is-large"
        max-height="450">

        <template slot-scope="props">
          <div class="media">
            <div class="media-left">
              <vue-load-image>
                <img slot="image" :src="getImageURL(props.option.cover_url)" width="80"/>
                <img slot="preloader" src="/ui/images/blank.png" width="80"/>
                <img slot="error" src="/ui/images/blank.png" width="80"/>
              </vue-load-image>
            </div>
            <div class="media-content">
              {{ props.option.site}}<br/>
              <div class="truncate"><strong>{{ props.option.title }}</strong></div>
              <div style="margin-top:0.5em">
                <small>
                  <span v-for="(c, idx) in props.option.cast" :key="'cast' + idx">
                    {{c.name}}<span v-if="idx < props.option.cast.length-1">, </span>
                  </span>
                </small>
              </div>
            </div>
            <div class="media-right">
              {{format(parseISO(props.option.release_date), "yyyy-MM-dd")}}
            </div>
          </div>
        </template>
      </b-autocomplete>
    </b-field>
  </b-modal>
</template>

<script>
import ky from 'ky'
import VueLoadImage from 'vue-load-image'
import GlobalEvents from 'vue-global-events'
import { format, parseISO } from 'date-fns'

export default {
  name: 'ModalNewTag',
  props: {
    active: Boolean,
    sceneId: String
  },
  components: { VueLoadImage, GlobalEvents },
  computed: {
    isActive: {
      get () {
        const out = this.$store.state.overlay.showQuickFind
        if (out === true) {
          this.$nextTick(() => {
            this.$refs.autocompleteInput.$refs.input.focus()
          })
        }
        return out
      },
      set (values) {
        this.$store.state.overlay.showQuickFind = values
      }
    }
  },
  data () {
    return {
      data: [],
      dataNumRequests: 0,
      dataNumResponses: 0,
      selected: null,
      isFetching: false
    }
  },
  methods: {
    format,
    parseISO,
    getAsyncData: async function (query) {
      const requestIndex = this.dataNumRequests
      this.dataNumRequests = this.dataNumRequests + 1

      if (!query.length) {
        this.data = []
        this.dataNumResponses = requestIndex + 1
        this.isFetching = false
        return
      }

      this.isFetching = true

      const resp = await ky.get('/api/scene/search', {
        searchParams: {
          q: query
        }
      }).json()


      if (requestIndex >= this.dataNumResponses) {
        this.dataNumResponses = requestIndex + 1
        if (this.dataNumResponses === this.dataNumRequests) {
          this.isFetching = false
        }

        if (resp.results > 0) {
          this.data = resp.scenes
        } else {
          this.data = []
        }
      }
    },
    getImageURL (u) {
      if (u.startsWith('http')) {
        return '/img/120x/' + u.replace('://', ':/')
      } else {
        return u
      }
    },
    showSceneDetails (scene) {
      if (this.$router.currentRoute.name !== 'scenes') {
        this.$router.push({ name: 'scenes' })
      }
      this.$store.commit('overlay/hideQuickFind')
      this.$store.commit('overlay/showDetails', { scene })
    }
  }
}
</script>

<style scoped>
  .modal {
    justify-content: normal;
    padding-top: 9em;
  }

  .queryInput {
    width: 960px;
  }

  .truncate {
    width: 320px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
</style>
