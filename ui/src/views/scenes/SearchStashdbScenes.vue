<template>
  <b-modal :active="isModalActive"           
           has-modal-card
           trap-focus
           aria-role="dialog"
           @cancel="close"
           aria-modal
           can-cancel>
    

    <div class="modal-card" :style="getOverlayPosition()">
      <header class="modal-card-head">
        <p class="modal-card-title">Search Stashdb Scenes</p>
        <button class="delete" @click="close" aria-label="close"></button>
      </header>

      <div class="modal-card-body">
                <div >
          <b-field label="Find scene...">
            <b-input v-model="queryString" placeholder="Find scene..." @input="debouncedSearch" :loading="isFetching" custom-class="is-large"/>
          </b-field>
    
        <b-table :data="searchResults" >
          <b-table-column field="ImageUrl" >
            <template slot-scope="props">
              <div class="media">
                <div class="media-left">
                    <vue-load-image>
                        <img slot="image" :src="getImageURL(props.row.ImageUrl)" width="150" @mouseover="showTooltipImage(props.row.ImageUrl)" @mouseout="showTooltipImage('')" />
                        <img slot="preloader" src="/ui/images/blank.png" height="150"/>
                        <img slot="error" src="/ui/images/blank.png" height="150"/>
                    </vue-load-image>
                    <div v-if="tooltipImage!='' && tooltipImage==props.row.ImageUrl" class="tooltipimg">
                      <img :src="tooltipImage" alt="Tooltip Image" width="400px" />
                    </div>
                  <div v-if="props.row.Date!=''"><small><strong>Released:</strong> {{format(parseISO(props.row.Date), "yyyy-MM-dd")}}</small></div>
                  <div v-if="props.row.Duration!=''"><small><strong>Durn:</strong> {{ props.row.Duration }}</small></div>
                  <div><small><strong>Score:</strong> {{ props.row.Weight }}</small></div>
                  <div>
                    <a class="button is-primary is-small" @click="linktoStashdb(props.row)" :title="'Link scene with stashdb'">
                      <b-icon pack="mdi" :icon="'link-variant-plus'" size="is-small"/>
                    </a>
                  </div>
                </div>
                <div class="media-content">
                  <div class="truncate"><strong><a :href="props.row.Url"  target="_blank">{{ props.row.Studio }} - {{ props.row.Title }}</a></strong></div>
                  <div><small style="white-space: normal; display: block;">{{props.row.Description}}</small></div>
                  <div style="margin-top:0.5em">
                    <small style="white-space: normal; display: block;">
                      <span v-for="(c, idx) in props.row.Performers" :key="'Performers' + idx">{{c.Name}}<span v-if="idx < props.row.Performers.length-1">, </span></span>
                    </small>
                  </div>
                </div>            
              </div>
            </template>
          </b-table-column>
        </b-table>
        </div>
      </div>
      <footer class="modal-card-foot">
      </footer>
    </div>
  </b-modal>
</template>

<script>
import GlobalEvents from 'vue-global-events'
import ky from 'ky'
import VueLoadImage from 'vue-load-image'
import { format, parseISO } from 'date-fns'

function debounce(func, wait) {
  let timeout;
  return function(...args) {
    const context = this;
    clearTimeout(timeout);
    timeout = setTimeout(() => func.apply(context, args), wait);
  };
}

export default {
  name: 'SearchStashdbScenes',
  components: {  GlobalEvents, VueLoadImage },
  data () {
    return {
        isModalActive: true,
        stashdbUrl: "",
        searchResults: [],
        queryString: "",
        isFetching: false,
        tooltipImage: '',
        scene: "",
        }        
  },
  created() {
    this.debouncedSearch = debounce(this.searchStashdb, 750); // 750ms delay
  },
  mounted () {
    const item = Object.assign({}, this.$store.state.overlay.searchStashDbScenes.scene)    
    this.scene = item
    this.openDialog(item)    
  },
  methods: {
    format,
    parseISO,
    close () {
      this.$store.commit('overlay/hideSearchStashdbScenes')
    },
    searchStashdb() {
        this.$buefy.toast.open({message: `Searching scenes`, type: 'is-primary', duration: 5000})
        ky.get('/api/extref/stashdb/search/' + this.scene.id + "?q=" + this.queryString, {timeout: 6e6}).json().then(data => {
            this.searchResults = Object.values(data.Results).sort((a, b) => b.Weight - a.Weight)
            this.isModalActive = true
            if (data.Status!='') {
              this.$buefy.toast.open({message: `Warning:  ${data.Status}`, type: 'is-warning', duration: 5000})
            }
        })
    },
    selectScene(option) {
        this.stashdbUrl=option.Url.replace("https://stashdb.org/scenes/","")
        this.$nextTick(() => {
            if (this.$refs.autocompleteInput) {
                this.$refs.autocompleteInput.focus();
            }
        });
    },    
    linktoStashdb(option) {
        this.stashdbUrl=option.Url.replace("https://stashdb.org/scenes/","")
        ky.get('/api/extref/stashdb/link2scene/' + this.scene.id +'/'+this.stashdbUrl ).json().then(data => {          
          this.$store.commit('sceneList/updateScene', data)
          this.$store.commit('overlay/showDetails', { scene: data })
          this.close()
        })
    },    
    getImageURL (u) {        
      if (u != undefined && u.startsWith('http')) {
        return '/img/120x/' + u.replace('://', ':/')
      } else {
        return u
      }
    },
    openDialog(scene) {
        this.isModalActive = true
        this.searchStashdb()
        this.$store.commit('overlay/changeDetailsTab', { tab: 3 })
        this.$nextTick(() => {
            if (this.$refs.autocompleteInput) {
                this.$refs.autocompleteInput.focus();
            }
        });
        this.scene = scene
    },
    getOverlayPosition(){
      if (this.$store.state.overlay.searchStashDbScenes.scene.synopsis == "") {
        return "height: 65vh; width: 40vw; left: 20vw; top: 20vh;"
      } else {
        return "height: 65vh; width: 40vw; left: -20vw"
      }
    },
    showTooltipImage(val){
      this.tooltipImage=val
    },
  },
  computed: {

}
}
</script>

<style scoped>
.b-modal {
  left: -20%;
  width: 40%;
  height: 65%;
  overflow: auto;
}

.tab-item {
  height: 40vh;
}
.tooltipimg {
  position: absolute;
  z-index: 1;
  width: 350;
  background-color: white;
  box-shadow: 0 2px 5px rgba(0, 0, 0, 0.3);
  display: flex;
  justify-content: center;
  align-items: center;
  padding: 10px;
  transform: translateX(60px) translateY(-50px);
}
.tooltipimg img {
  max-width: 100%;
  max-height: 100%;
}
</style>
