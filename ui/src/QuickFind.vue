<template>
  <b-modal :active.sync="isActive"
           :destroy-on-hide="false"
           has-modal-card
           trap-focus
           aria-role="dialog"
           aria-modal
           can-cancel>
    <b-field grouped>
      <b-tooltip :label="$t('Optional: select one or more words to target searching to a specific field')" :delay="500" position="is-right">
        <b-taglist>
          <b-tag class="tag is-info is-small">{{$t('Search Fields')}}</b-tag>
          <b-button @click='searchPrefix("title:")' class="tag is-info is-small is-light">title:</b-button>
          <b-button @click='searchPrefix("cast:")' class="tag is-info is-small is-light">cast:</b-button>
          <b-button @click='searchPrefix("site:")' class="tag is-info is-small is-light">site:</b-button>
          <b-button @click='searchPrefix("id:")' class="tag is-info is-small is-light">id:</b-button>
          <b-button @click='searchDurationPrefix("duration:")' class="tag is-info is-small is-light">duration:</b-button>
          <b-tooltip :label="$t('Defaults date range to the last week. Note:must match yyyy-mm-dd, include leading zeros')" :delay="500" position="is-top">
            <b-button @click='searchDatePrefix("released:")' class="tag is-info is-small is-light">released:</b-button>
            <b-button @click='searchDatePrefix("added:")' class="tag is-info is-small is-light">added:</b-button>
          </b-tooltip>
        </b-taglist>
      </b-tooltip>
    </b-field>
    <b-field style="width:600px">
      <b-autocomplete
        ref="autocompleteInput"
        :data="data"
        placeholder="Find scene..."
        field="query"
        :loading="isFetching"
        v-model="queryString"
        @typing="getAsyncData"
        @select="option => showSceneDetails(option)"
        :open-on-focus="true"
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
              {{ props.option.site}} 
              <b-icon v-if="props.option.is_hidden" pack="mdi" icon="eye-off-outline" size="is-small"/><br/>
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
      isFetching: false,
      queryString: ""
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
      this.data = []
      this.$store.commit('overlay/showDetails', { scene })
    },
    searchPrefix(prefix) {      
      let textbox = this.$refs.autocompleteInput.$refs.input.$refs.input
      if (textbox.selectionStart != textbox.selectionEnd) {
        let selected = textbox.value.substring(textbox.selectionStart, textbox.selectionEnd)
        selected=selected.replace(/_/g," ").replace(/-/g," ").trim()
        if (selected.indexOf(' ') >= 0) {
          selected='"' + selected + '"'
        }
        this.queryString = textbox.value.substring(0,textbox.selectionStart) + " " + prefix + selected + " " + textbox.value.substr(textbox.selectionEnd)
        this.getAsyncData(this.queryString)
        this.$refs.autocompleteInput.focus()
      }
    },
    searchDatePrefix(prefix) {      
        let today = new Date().toISOString().slice(0, 10)
        let weekago = new Date(Date.now() - 604800000).toISOString().slice(0, 10)
        if (this.queryString == undefined) {
          this.queryString = prefix + '>="' + weekago + '" ' +  prefix + '<="' + today + '"'          
        } else {
          this.queryString = this.queryString.trim() + ' ' + prefix + '>="' + weekago + '" ' +  prefix + '<="' + today + '"'        
        }
        this.getAsyncData(this.queryString)
        this.$refs.autocompleteInput.focus()
    },
    searchDurationPrefix(prefix) {
      if (this.queryString == undefined) {
        this.queryString = prefix + '>=0'
      } else {
        this.queryString = this.queryString.trim() + ' ' + prefix + '>=0'
      }
      this.getAsyncData(this.queryString)
      this.$refs.autocompleteInput.focus()
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
