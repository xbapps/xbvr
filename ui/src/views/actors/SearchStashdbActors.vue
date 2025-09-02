<template>
    <b-modal :active="isModalActive"           
           has-modal-card
           trap-focus
           aria-role="dialog"
           @cancel="close"
           aria-modal
           can-cancel>
    
    <GlobalEvents
      :filter="e => !['INPUT', 'TEXTAREA'].includes(e.target.tagName)"
      @keyup.esc="close"
      @keydown.k="handleLeftArrow"
      @keydown.l="handleRightArrow"
    />

    <div class="modal-card" style="height: 80vh; width: 60vw; left: 10vw">
      <header class="modal-card-head">
        <p class="modal-card-title">Search Stashdb Actors</p>
        <button class="delete" @click="close" aria-label="close"></button>
      </header>

      <div class="modal-card-body">
        <div >
          <b-field label="Find actor...">
            <b-input v-model="queryString" placeholder="Find actor..." @input="debouncedSearch" :loading="isFetching" custom-class="is-large"/>
          </b-field>
    
        <b-table :data="searchResults" @click="onRowSelected" >
          <b-table-column field="Name" >
            <template slot-scope="props">
              <div class="media">
                <div class="media-left">
                  <b-carousel 
                    :ref="'actorcarousel' + props.index"
                    :id="'actorcarousel' + props.index"
                    :autoplay="false" :indicator="false" icon-size="is-small" width="120"
                  >
                    <b-carousel-item v-for="(image, index) in props.row.ImageUrl" :key="index">
                          <vue-load-image height="50px">
                            <img slot="image" :src="image && image.length ? getImageURL(image) : '/ui/images/blank_female_profile.png'" width="100"  @mouseover="setShowTooltipImage(image, props.row.Id)" @mouseout="setShowTooltipImage('','')"/>
                            <img slot="preloader" src="/ui/images/blank.png" width="100" />
                            <img slot="error" src="/ui/images/blank.png" width="100" />
                          </vue-load-image>
                    </b-carousel-item>
                  </b-carousel>
                    <div v-if="tooltipImage!='' && tooltipID==props.row.Id" class="tooltipimg" @mouseout="setShowTooltipImage('','')">                      
                          <vue-load-image width="300">
                            <img slot="image" :src="tooltipImage" width="300"  />
                            <img slot="preloader" src="/ui/images/blank.png" width="300" />
                            <img slot="error" src="/ui/images/blank.png" width="300" />
                          </vue-load-image>
                    </div>
                  <div v-if="props.row.DOB">
                    <span class="smaller-text">
                      <strong>Birth Date:</strong>
                    </span>
                  </div>
                  <div v-if="props.row.DOB">
                    <span class="smaller-text">{{ format(parseISO(props.row.DOB), "yyyy-MM-dd") }}</span>
                  </div>
                  <div>
                    <span class="smaller-text">
                      <strong>Score:</strong> {{ props.row.Weight }}
                    </span>
                  </div>
                  <div>
                    <a class="button is-primary is-small" @click="linktoStashdb(props.row)" :title="'Link Actor with stashdb'">
                      <b-icon pack="mdi" :icon="'link-variant-plus'" size="is-small" />
                    </a>
                  </div>
                </div>
                <div class="media-content">
                  <div class="truncate">
                    <strong>
                      <a :href="props.row.Url" target="_blank">{{ props.row.Name }} - {{ props.row.Disambiguation }}</a>
                    </strong>
                  </div>
                  <div>
                    <strong>Aliases:</strong>
                    <b-tag v-for="alias in props.row.Aliases" :key="alias.Alias" :class="{ 'is-primary': alias.Matched }"  style="margin-right: 2px;"> {{ alias.Alias }}</b-tag>
                  </div>
                  <div>                    
                    <b-tag v-for="link in props.row.Studios" :key="link.url"><a :href="link.Url" :class="{ 'bold-tag': link.Matched }" target="_blank" style="margin-right: 2px;">{{ link.Name }}({{ link.SceneCount }})</a></b-tag>
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
  name: 'SearchStashdbActors',
  components: {  GlobalEvents, VueLoadImage },
  data () {
    return {
        isModalActive: true,
        stashdbUrl: "",
        searchResults: [],
        queryString: "",
        isFetching: false,
        tooltipImage: "",
        tooltipID: "",
        actor: "",
        selectedRow: undefined,
        }        
  },
  created() {
    this.debouncedSearch = debounce(this.searchStashdb, 750); // 750ms delay
  },
  mounted () {
    const item = Object.assign({}, this.$store.state.overlay.searchStashDbActors.actor)    
    this.actor = item
    this.openDialog(item)
    this.queryString=this.actor.name
  },
  methods: {
    format,
    parseISO,
    close () {
      this.$store.commit('overlay/hideSearchStashdbActors')
    },
    searchStashdb() {
      this.$buefy.toast.open({message: `Searching Actors`, type: 'is-primary', duration: 5000})
        ky.get('/api/extref/stashdb/searchactor/' + this.actor.id + "?q=" + this.queryString, {timeout: 6e6}).json().then(data => {
            this.searchResults = Object.values(data.Results).sort((a, b) => b.Weight - a.Weight)
            this.isModalActive = true
            if (data.Status!='') {
              this.$buefy.toast.open({message: `Warning:  ${data.Status}`, type: 'is-warning', duration: 5000})
            }
        })
    },
    selectActor(option) {
        this.stashdbUrl=option.Url.replace("https://stashdb.org/performers/","")
        this.$nextTick(() => {
            if (this.$refs.autocompleteInput) {
                this.$refs.autocompleteInput.focus();
            }
        });
    },    
    linktoStashdb(option) {
        this.stashdbUrl=option.Url.replace("https://stashdb.org/performers/","")
        ky.get('/api/extref/stashdb/link2actor/' + this.actor.id +'/'+this.stashdbUrl ).json().then(data => {          
          // this.$store.commit('sceneList/updateScene', data)
           this.$store.commit('overlay/showActorDetails', { actor: data })
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
    openDialog(actor) {
        this.isModalActive = true
        this.searchStashdb()
        this.$nextTick(() => {
            if (this.$refs.autocompleteInput) {
                this.$refs.autocompleteInput.focus();
            }
        });
        this.actor = actor
    },
    setShowTooltipImage(val, id){
      this.tooltipImage=val
      this.tooltipID=id
    },
    handleLeftArrow() {
      let idx=0
      if (this.selectedRow!=undefined && this.searchResults.length){
        idx=this.searchResults.findIndex(element => element.Id==this.selectedRow.Id) 
      }
      let selectedCarousel = this.$refs['actorcarousel' + idx]
      selectedCarousel.prev()
},
    handleRightArrow() {
      let idx=0
      if (this.selectedRow!=undefined && this.searchResults.length){
        idx=this.searchResults.findIndex(element => element.Id==this.selectedRow.Id) 
      }
      let selectedCarousel = this.$refs['actorcarousel' + idx]
      selectedCarousel.next()
    },
    onRowSelected(row) {
      this.selectedRow= row      
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
  transform: translateX(60px) translateY(-50px);}
.tooltipimg img {
  max-width: 100%;
  max-height: 100%;
}
.smaller-text {
  font-size: 0.8em; /* or any smaller size you prefer */
}
.bold-tag {
  font-weight: bold;
}
</style>
