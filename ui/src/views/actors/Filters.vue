<template>
  <div>
    <div class="is-divider" data-content="Saved searches" style="margin-top:0.8em;"></div>

     <SavedSearch/> 

    <div class="is-divider" data-content="Properties"></div>

    <div class="columns is-multiline is-gapless">
      <div class="column is-half">
        <b-checkbox-button v-model="lists" native-value="watchlist" type="is-primary">
          <b-icon pack="mdi" icon="calendar-check"/>
          <span>{{ $t('Watchlist') }}</span>
        </b-checkbox-button>
      </div>
      <div class="column is-half">
        <b-checkbox-button v-model="lists" native-value="favourite" type="is-danger">
          <b-icon pack="mdi" icon="heart"/>
          <span>{{ $t('Favourite') }}</span>
        </b-checkbox-button>
      </div>
    </div>

    <div class="is-divider" data-content="Sorting"></div>

    <b-field :label="$t('Sort by')" label-position="on-border" :addons="true" class="field-extra">
      <div class="control is-expanded">
        <div class="select is-fullwidth">
          <select v-model="sort">
            <option value="name_asc">↑ {{ $t("Name") }}</option>
            <option value="name_desc">↓ {{ $t("Name") }}</option>
            <option value="birthday_desc">↓ {{ $t("Birthdate") }}</option>
            <option value="birthday_asc">↑ {{ $t("Birthdate") }}</option>
            <option value="rating_desc">↓ {{ $t("Rating") }}</option>
            <option value="rating_asc">↑ {{ $t("Rating") }}</option>
            <option value="scene_rating_desc">↓ {{ $t("Average Scene Rating") }}</option>
            <option value="added_desc">↓ {{ $t("Added date") }}</option>
            <option value="added_asc">↑ {{ $t("Added date") }}</option>
            <option value="modified_desc">↓ {{ $t("Modified date") }}</option>
            <option value="modified_asc">↑ {{ $t("Modified date") }}</option>
            <option value="scene_release_desc">↓ {{ $t("Scene Released Date") }}</option>
            <option value="scene_added_desc">↓ {{ $t("Scene Added Date") }}</option>
            <option value="file_added_desc">↓ {{ $t("File Added Date") }}</option>
            <option value="random">↯ {{ $t("Random") }}</option>
          </select>
        </div>
      </div>
    </b-field>

    <div class="is-divider" data-content="Actor Filters"></div>

    <div v-if="Object.keys(filters).length !== 0">
      <b-field :label="$t('Cast')" label-position="on-border" class="field-extra">
        <b-taginput v-model="cast" autocomplete :data="filteredCast" @typing="getFilteredCast">
          <template slot-scope="props">{{ props.option }}</template>
          <template slot="empty">{{ $t("No matching cast") }}</template>
          <template #selected="props">
              <b-tag v-for="(tag, index) in props.tags"
                :type="tag.charAt(0)=='!' ? 'is-danger': (tag.charAt(0)=='&' ? 'is-success' : '')"
                :key="tag+index" :tabstop="false" closable  @close="cast=cast.filter(e => e !== tag)" @click="toggle2Way(tag,index,'cast')">              
                <b-tooltip position="is-right" :delay="200"
                  :label="tag.charAt(0)=='!' ? 'Exclude ' + removeConditionPrefix(tag) : tag.charAt(0)=='&' ? 'Must Have ' + removeConditionPrefix(tag) : 'Include ' + removeConditionPrefix(tag)">
                  <b-icon pack="mdi" v-if="tag.charAt(0)=='!'" icon="minus-circle-outline" size="is-small" class="tagicon"></b-icon>
                  <b-icon pack="mdi" v-if="tag.charAt(0)=='&'" icon="plus-circle-outline" size="is-small" class="tagicon"></b-icon>
                  {{removeConditionPrefix(tag)}}
                </b-tooltip>
              </b-tag>
          </template>
        </b-taginput>
      </b-field>

      <b-field :label="$t('Site')" label-position="on-border" class="field-extra">
        <b-taginput v-model="sites" autocomplete :data="filteredSites" @typing="getFilteredSites">
          <template slot-scope="props">{{ props.option }}</template>
          <template slot="empty">{{ $t("No matching sites") }}No matching sites</template>
          <template #selected="props">
            <b-tag v-for="(tag, index) in props.tags"
              :type="tag.charAt(0)=='!' ? 'is-danger': (tag.charAt(0)=='&' ? 'is-success' : '')"
              :key="tag+index" :tabstop="false" closable  @close="sites=sites.filter(e => e !== tag)" @click="toggle3way(tag,index,'sites')">
                <b-tooltip position="is-right" :delay="200"
                  :label="tag.charAt(0)=='!' ? 'Exclude ' + removeConditionPrefix(tag) : 'Include ' + removeConditionPrefix(tag)">
                  <b-icon pack="mdi" v-if="tag.charAt(0)=='!'" icon="minus-circle-outline" size="is-small" class="tagicon"></b-icon>
                  {{removeConditionPrefix(tag)}}
                </b-tooltip>
            </b-tag>
          </template>
        </b-taginput>
      </b-field>

      <b-tooltip position="is-top" :label="$t('Allows searching a variety of attributes such as: Possible Aka actors, Cup Size, Eye/Hair Color, Has Tattoo, Has Piercing, Breast Type, Nationailty, Ethnicity, Aka, Has Images')" multilined :delay="1000" style="width:100%">
        <b-field :label="$t('Attributes')" label-position="on-border" class="field-extra">
          <b-taginput v-model="attributes" autocomplete :data="filteredAttributes" @typing="getFilteredAttributes">
            <template slot-scope="props">{{ props.option }}</template>
            <template slot="empty">{{ $t("No matching attributes") }}</template>
            <template #selected="props">
              <b-tag v-for="(tag, index) in props.tags"
                :type="tag.charAt(0)=='!' ? 'is-danger': (tag.charAt(0)=='&' ? 'is-success' : '')"
                :key="tag+index" :tabstop="false" closable  @close="attributes=attributes.filter(e => e !== tag)" @click="toggle3way(tag,index,'attributes')"> 
                  <b-icon pack="mdi" v-if="tag.charAt(0)=='!'" icon="minus-circle-outline" size="is-small" class="tagicon"></b-icon>
                  <b-icon pack="mdi" v-if="tag.charAt(0)=='&'" icon="plus-circle-outline" size="is-small" class="tagicon"></b-icon>
                  {{removeConditionPrefix(tag)}}
              </b-tag>
            </template>          
          </b-taginput>
        </b-field>
      </b-tooltip>

      <table width="100%">
        <tr>
          <td class="slider-title"><strong><small>{{ $t("Age") }}:</small></strong></td>
          <td ><b-slider :min="18" :max="100" :step="1" :tooltip="true" v-model="ages" lazy class="slider"></b-slider></td>
        </tr>
        <tr>
          <td class="slider-title"><strong><small>{{ $t("Height") }}:</small></strong></td>
          <td><b-slider :min="120" :max="220" :step="1" :tooltip="true" v-model="heights" lazy class="slider"></b-slider></td>
        </tr>
        <tr>
          <td class="slider-title"><strong><small>{{ $t("Weight") }}:</small></strong></td>
          <td><b-slider :min="25" :max="150" :step="1" :tooltip="true" v-model="weights" lazy class="slider"></b-slider></td>
        </tr>
        <tr>
          <td class="slider-title"><strong><small>{{ $t("Scenes") }}:</small></strong></td>
          <td><b-slider :min="0" :max="150" :step="1" :tooltip="true" v-model="scenecounts" lazy class="slider" ></b-slider></td>
        </tr>
        <tr>
          <td class="slider-title"><strong><small>{{ $t("Available") }}:</small></strong></td>
          <td><b-slider :min="0" :max="150" :step="1" :tooltip="true" v-model="avails" lazy class="slider" ></b-slider></td>
        </tr>
        <tr>
          <td class="slider-title"><strong><small>{{ $t("Rating") }}:</small></strong></td>
          <td><b-slider :min="0" :max="5" :step=".5" :tooltip="true" v-model="ratings" lazy class="slider" ></b-slider></td>
        </tr>
        <tr>
          <td class="slider-title"><strong><small>{{ $t("Scene Rating") }}:</small></strong></td>
          <td><b-slider :min="0" :max="5" :step=".25" :tooltip="true" v-model="sceneratings" lazy class="slider" ></b-slider></td>
        </tr>   
      </table>
    </div>
    <div class="is-divider" data-content="Actor Also Known As groups"></div>
    <b-field>
      <b-tooltip position="is-right" :label="$t('New Aka Group. Select 2 or more actors in the Cast filter')" multilined :delay="200">
        <button class="button is-small is-outlined" @click="createAkaGroup" :disabled="disableNewAkaGroup">
          <b-icon pack="mdi" icon="account-multiple-plus-outline"></b-icon>
        </button>
      </b-tooltip>
      <b-tooltip position="is-right" :label="$t('Select the Aka Group to delete in the Cast Filter')" multilined :delay="200">
        <button class="button is-small is-outlined" @click="deleteAkaGroup" :disabled="disableDeleteAkaGroup">
          <b-icon pack="mdi" icon="delete-outline"></b-icon>
        </button>
      </b-tooltip>
      <b-tooltip position="is-bottom" :label="$t('Add Cast to Aka Group. Select the Aka group and Actors to add in the Cast Filter')" multilined :delay="200">
        <button class="button is-small is-outlined" @click="addToAkaGroup" :disabled="disableAddToAkaGroup">
          <b-icon pack="mdi" icon="account-plus-outline"></b-icon>
        </button>
      </b-tooltip>
      <b-tooltip position="is-bottom" :label="$t('Remove Cast from Aka Group. Select the Aka group and Actors to remove in the Cast Filter')" multilined :delay="200">
        <button class="button is-small is-outlined" @click="removeFromAkaGroup" :disabled="disableRemoveFromAkaGroup">
          <b-icon pack="mdi" icon="account-minus-outline"></b-icon>
        </button>
      </b-tooltip>

    </b-field>
  </div>
</template>

<script>
import SavedSearch from './SavedSearch'
import ky from 'ky'

export default {
  name: 'Filters',
  components: { SavedSearch },
  mounted () {
    this.$store.dispatch('actorList/filters')
    this.fetchFilters()
  },
  data () {
    return {
      filteredCast: [],
      filteredSites: [],
      filteredTags: [],
      filteredAttributes: [],
    }
  },
  methods: {
    reloadList () {
      this.$router.push({
        name: 'actors',
        query: {
          q: this.$store.getters['actorList/filterQueryParams']
        }
      })
    },
    getFilteredCast (text) {
      // Load cast matching at the start 
      let startWithText = this.filters.cast.filter(option => (
        (option.toString().toLowerCase().indexOf(text.toLowerCase()) == 0 || option.toString().toLowerCase().indexOf("aka:" + text.toLowerCase()) == 0 ) &&
        !this.cast.some(entry => this.removeConditionPrefix(entry.toString()) === option.toString())
      ))
      // Now load actors containing the text
      let containsText = this.filters.cast.filter(option => (
        option.toString().toLowerCase().indexOf(text.toLowerCase()) > 0 &&
        !this.cast.some(entry => this.removeConditionPrefix(entry.toString()) === option.toString())
      ))
      this.filteredCast = startWithText.concat(containsText)
    },
    getFilteredSites (text) {
      this.filteredSites = this.filters.sites.filter(option => (
        option.toString().toLowerCase().indexOf(text.toLowerCase()) >= 0 &&
        !this.sites.some(entry => this.removeConditionPrefix(entry.toString()) === option.toString())
      ))
    },
    getFilteredAttributes (text) {
      this.filteredAttributes = this.filters.attributes.filter(option => (
        option.toString().toLowerCase().indexOf(text.toLowerCase()) >= 0 &&
        !this.attributes.some(entry => this.removeConditionPrefix(entry.toString()) === option.toString())
      ))
    },
    createAkaGroup () {
      this.$store.state.actorList.isLoading = true
      ky.post('/api/aka/create', {json: {actorList: this.cast}}).json().then(data => {
        this.cast.push(data.akas.aka_actor.name)
        this.$store.dispatch('actorList/filters')
        this.reloadList()
        if (data.status != '') {
          this.$buefy.toast.open({message: `Warning:  ${data.status}`, type: 'is-warning', duration: 5000})
        }
        this.$store.state.actorList.isLoading = false
      })
    },
    deleteAkaGroup () {
      this.$store.state.actorList.isLoading = true
      ky.post('/api/aka/delete', {json: {name: this.cast[0]}}).json().then(data => {
        this.cast = []
        this.$store.dispatch('actorList/filters')
        this.reloadList()
        this.$store.state.actorList.isLoading = false
      })       
    },
    addToAkaGroup () {
      this.$store.state.actorList.isLoading = true
      ky.post('/api/aka/add', {json: {actorList: this.cast}}).json().then(data => {        
        // delete old aka & add new name
        this.cast = this.cast.filter(e => !e.startsWith("aka:")) 
        this.cast.push(data.akas.aka_actor.name) 
        this.$store.dispatch('actorList/filters')       
        this.reloadList()        
        if (data.status != '') {
          this.$buefy.toast.open({message: `Warning:  ${data.status}`, type: 'is-warning', duration: 5000})
        }
        this.$store.state.actorList.isLoading = false
      })
      
    },
    removeFromAkaGroup () {
      this.$store.state.actorList.isLoading = true
      ky.post('/api/aka/remove', {json: {actorList: this.cast}}).json().then(data => {        
        // delete old aka & add new name
        this.cast = this.cast.filter(e => !e.startsWith("aka:")) 
        this.cast.push(data.akas.aka_actor.name)
        this.$store.dispatch('actorList/filters')
        this.reloadList()
        if (data.status != '') {
          this.$buefy.toast.open({message: `Warning:  ${data.status}`, type: 'is-warning', duration: 5000})
        }
        this.$store.state.actorList.isLoading = false
      })
    },
    toggle3way (text, idx, list) {      
      let tags = []
      switch (list) {
        case 'cast':
          tags=this.cast 
          break
        case 'sites':
          tags=this.sites
          break
        case 'attributes':
          tags=this.attributes
          break
      }      
      switch(tags[idx].charAt(0)) {
        case '!':
          tags[idx]=this.removeConditionPrefix(tags[idx])
          break
        case '&':
          tags[idx]='!' + this.removeConditionPrefix(tags[idx])
        break
        default:
        tags[idx]='&'+text        
      }      
      switch (list) {
        case 'cast':
          this.cast=tags
          break
        case 'sites':
          this.sites=tags
          break
        case 'attributes':
          this.attributes=tags
          break
      }
    },    
    toggle2Way (text, idx, list) {      
      let tags = []
      switch (list) {
        case 'sites':
          tags=this.sites
          break
        case 'cast':
          tags=this.cast
          break
      }      
      switch(tags[idx].charAt(0)) {
        case '!':
          tags[idx]=this.removeConditionPrefix(tags[idx])
          break
        default:
        tags[idx]='!'+text        
      }      
      switch (list) {
        case 'sites':
          this.sites=tags
          break
        case 'cast':
          this.cast=tags
          break
      }      
    },    
    removeConditionPrefix(txt) {
      if (txt.charAt(0)=='!' || txt.charAt(0)=='&') {
        return txt.substring(1) 
      }
      return txt
    },
    async fetchFilters() {
        this.filteredAttributes=['Loading attributes']
        ky.get('/api/actor/filters').json().then(data => {
          this.filteredAttributes=data.attributes          
      })      
    }
  },
  computed: {
    filters () {
      return this.$store.state.actorList.filterOpts
    },
    lists: {
      get () {
        return this.$store.state.actorList.filters.lists
      },
      set (value) {
        this.$store.state.actorList.filters.lists = value
        this.reloadList()
      }
    },
    cast: {
      get () {        
        return this.$store.state.actorList.filters.cast
      },
      set (value) {
        this.$store.state.actorList.filters.cast = value
        this.reloadList()
      }
    },
    sites: {
      get () {
        return this.$store.state.actorList.filters.sites
      },
      set (value) {
        this.$store.state.actorList.filters.sites = value
        this.reloadList()
      }
    },
    ages: {
      get () {
        return [this.$store.state.actorList.filters.min_age, this.$store.state.actorList.filters.max_age]
      },
      set (value) {
        if (this.$store.state.actorList.filters.min_age != value[0] || this.$store.state.actorList.filters.max_age != value[1]) {
          this.$store.state.actorList.filters.min_age = value[0]
          this.$store.state.actorList.filters.max_age = value[1]
          this.reloadList()
        }
      }
    },
    heights: {
      get () {
        return [this.$store.state.actorList.filters.min_height, this.$store.state.actorList.filters.max_height]
      },
      set (value) {
        if (this.$store.state.actorList.filters.min_height != value[0] || this.$store.state.actorList.filters.max_height != value[1]){
          this.$store.state.actorList.filters.min_height = value[0]
          this.$store.state.actorList.filters.max_height = value[1]
          this.reloadList()
        }
      }
    },
    weights: {
      get () {
        return [this.$store.state.actorList.filters.min_weight, this.$store.state.actorList.filters.max_weight]
      },
      set (value) {
        if (this.$store.state.actorList.filters.min_weight != value[0] || this.$store.state.actorList.filters.max_weight != value[1]){
          this.$store.state.actorList.filters.min_weight = value[0]
          this.$store.state.actorList.filters.max_weight = value[1]
          this.reloadList()
        }
      }
    },
    scenecounts: {
      get () {
        return [this.$store.state.actorList.filters.min_count, this.$store.state.actorList.filters.max_count]
      },
      set (value) {
        if (this.$store.state.actorList.filters.min_count != value[0] || this.$store.state.actorList.filters.max_count != value[1]){
          this.$store.state.actorList.filters.min_count = value[0]
          this.$store.state.actorList.filters.max_count = value[1]
          this.reloadList()
        }
      }
    },
    avails: {
      get () {
        return [this.$store.state.actorList.filters.min_avail, this.$store.state.actorList.filters.max_avail]
      },
      set (value) {
        if (this.$store.state.actorList.filters.min_avail != value[0] || this.$store.state.actorList.filters.max_avail != value[1]){
          this.$store.state.actorList.filters.min_avail = value[0]
          this.$store.state.actorList.filters.max_avail = value[1]
          this.reloadList()
        }
      }
    },
    ratings: {
      get () {
        return [this.$store.state.actorList.filters.min_rating, this.$store.state.actorList.filters.max_rating]
      },
      set (value) {
        if (this.$store.state.actorList.filters.min_rating != value[0] || this.$store.state.actorList.filters.max_rating != value[1]){
          this.$store.state.actorList.filters.min_rating = value[0]
          this.$store.state.actorList.filters.max_rating = value[1]
          this.reloadList()
        }
      }
    },
    sceneratings: {
      get () {
        return [this.$store.state.actorList.filters.min_scene_rating, this.$store.state.actorList.filters.max_scene_rating]
      },
      set (value) {
        if (this.$store.state.actorList.filters.min_scene_rating != value[0] || this.$store.state.actorList.filters.max_scene_rating != value[1]){
          this.$store.state.actorList.filters.min_scene_rating = value[0]
          this.$store.state.actorList.filters.max_scene_rating = value[1]
          this.reloadList()
        }
      }
    },
    attributes: {
      get () {
        return this.$store.state.actorList.filters.attributes
      },
      set (value) {
        this.$store.state.actorList.filters.attributes = value
        this.reloadList()        
      }
    },
    sort: {
      get () {
        return this.$store.state.actorList.filters.sort
      },
      set (value) {
        this.$store.state.actorList.filters.sort = value
        this.reloadList()
      }
    },
    disableNewAkaGroup() {
      let akaCastCnt = 0
      let actorCnt = 0       
 
      for (var i = 0; i < this.cast.length; i++) {        
        if (this.cast[i].startsWith("aka:")) {
          akaCastCnt++
        } else {
          actorCnt++
        }
      }
      // you can create a new group from a list of actors (more than one)
      return akaCastCnt == 0 && actorCnt > 1 ? false : true
    },
    disableDeleteAkaGroup() {
      let akaCastCnt = 0
      let actorCnt = 0       
 
      for (var i = 0; i < this.cast.length; i++) {        
        if (this.cast[i].startsWith("aka:")) {
          akaCastCnt++
        } else {
          actorCnt++
        }
      }

      // you can only delete a group when it is the only thing selected      
      return akaCastCnt == 1 && actorCnt == 0 > 1 ? false : true
    },
    disableAddToAkaGroup() {
      let akaCastCnt = 0
      let actorCnt = 0       
 
      for (var i = 0; i < this.cast.length; i++) {        
        if (this.cast[i].startsWith("aka:")) {
          akaCastCnt++
        } else {
          actorCnt++
        }
      }

      // you can add to a group if you select one group and one or more actors
      return akaCastCnt == 1 && actorCnt > 0 ? false : true
    },
    disableRemoveFromAkaGroup() {
      let akaCastCnt = 0
      let actorCnt = 0       
 
      for (var i = 0; i < this.cast.length; i++) {        
        if (this.cast[i].startsWith("aka:")) {
          akaCastCnt++
        } else {
          actorCnt++
        }
      }

      // you can remove from a group if you select one group and one or more actors
      return akaCastCnt == 1 && actorCnt > 0 ? false : true

    },
}
}
</script>

<style lang="scss" scoped>
@import "~bulma-extensions/bulma-divider/dist/css/bulma-divider.min.css";

.is-gapless div.control {
  margin: 0.1rem;
}

.is-divider {
  margin: 1.5rem 0;
}

.field-extra {
  margin-bottom: 1.1em !important;
}

.tagicon {
  margin-right: -0.2em !important;
}
.slider-title {
  width: 80px;
}
.slider {
  margin-right: "3em";
}
</style>
