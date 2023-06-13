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
      <div class="column is-half">
        <b-checkbox-button v-model="lists" native-value="scripted" type="is-info">
          <b-icon pack="mdi" icon="pulse"/>
          <span>{{ $t('Scripted') }}</span>
        </b-checkbox-button>
      </div>
    </div>

    <div class="is-divider" data-content="Sorting / Status / Release"></div>

    <b-field :label="$t('Sort by')" label-position="on-border" :addons="true" class="field-extra">
      <div class="control is-expanded">
        <div class="select is-fullwidth">
          <select v-model="sort">
            <option value="release_desc">↓ {{ $t("Release date") }}</option>
            <option value="release_asc">↑ {{ $t("Release date") }}</option>
            <option value="added_desc">↓ {{ $t("File added date") }}</option>
            <option value="added_asc">↑ {{ $t("File added date") }}</option>
            <option value="title_desc">↓ {{ $t("Title") }}</option>
            <option value="title_asc">↑ {{ $t("Title") }}</option>
            <option value="total_file_size_desc">↓ {{ $t("File size") }}</option>
            <option value="total_file_size_asc">↑ {{ $t("File size") }}</option>
            <option value="rating_desc">↓ {{ $t("Rating") }}</option>
            <option value="rating_asc">↑ {{ $t("Rating") }}</option>
            <option value="total_watch_time_desc">↓ {{ $t("Watch time") }}</option>
            <option value="total_watch_time_asc">↑ {{ $t("Watch time") }}</option>
            <option value="scene_added_desc">↓ {{ $t("Scene added date") }}</option>
            <option value="scene_updated_desc">↓ {{ $t("Scene updated date") }}</option>
            <option value="last_opened_desc">↓ {{ $t("Last viewed date") }}</option>
            <option value="last_opened_asc">↑ {{ $t("Last viewed date") }}</option>
            <option value="random">↯ {{ $t("Random") }}</option>
          </select>
        </div>
      </div>
    </b-field>

    <b-field label="Watched" label-position="on-border" :addons="true" class="field-extra">
      <div class="control is-expanded">
        <div class="select is-fullwidth">
          <select v-model="isWatched">
            <option :value="null">Everything</option>
            <option :value="true">Watched</option>
            <option :value="false">Unwatched</option>
          </select>
        </div>
      </div>
    </b-field>

    <b-field label="Release month" label-position="on-border" :addons="true" class="field-extra">
      <div class="control is-expanded">
        <div class="select is-fullwidth">
          <select v-model="releaseMonth">
            <option></option>
            <option v-for="t in filters.release_month" :key="t">{{ t }}</option>
          </select>
        </div>
      </div>
      <div class="control">
        <button type="submit" class="button is-light" @click="clearReleaseMonth">
          <b-icon pack="fas" icon="times" size="is-small"></b-icon>
        </button>
      </div>
    </b-field>

    <b-field label="Folder" label-position="on-border" :addons="true" class="field-extra">
      <div class="control is-expanded">
        <div class="select is-fullwidth">
          <select v-model="volume">
            <option :value="0"></option>
            <option v-for="t in filters.volumes" :key="t.id" :value="t.id">{{ t.path }}</option>
          </select>
        </div>
      </div>
      <div class="control">
        <button type="submit" class="button is-light" @click="clearVolume">
          <b-icon pack="fas" icon="times" size="is-small"></b-icon>
        </button>
      </div>
    </b-field>

    <div class="is-divider" data-content="Filters"></div>

    <div v-if="Object.keys(filters).length !== 0">
      <b-field label="Cast" label-position="on-border" class="field-extra">
        <b-taginput v-model="cast" autocomplete :data="filteredCast" @typing="getFilteredCast">
          <template slot-scope="props">{{ props.option }}</template>
          <template slot="empty">No matching cast</template>
          <template #selected="props">
              <b-tag v-for="(tag, index) in props.tags"
                :type="tag.charAt(0)=='!' ? 'is-danger': (tag.charAt(0)=='&' ? 'is-success' : '')"
                :key="tag+index" :tabstop="false" closable  @close="cast=cast.filter(e => e !== tag)" @click="toggle3way(tag,index,'cast')">              
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

      <b-field label="Site" label-position="on-border" class="field-extra">
        <b-taginput v-model="sites" autocomplete :data="filteredSites" @typing="getFilteredSites">
          <template slot-scope="props">{{ props.option }}</template>
          <template slot="empty">No matching sites</template>
          <template #selected="props">
            <b-tag v-for="(tag, index) in props.tags"
              :type="tag.charAt(0)=='!' ? 'is-danger': (tag.charAt(0)=='&' ? 'is-success' : '')"
              :key="tag+index" :tabstop="false" closable  @close="sites=sites.filter(e => e !== tag)" @click="toggle2Way(tag,index,'sites')">
                <b-tooltip position="is-right" :delay="200"
                  :label="tag.charAt(0)=='!' ? 'Exclude ' + removeConditionPrefix(tag) : 'Include ' + removeConditionPrefix(tag)">
                  <b-icon pack="mdi" v-if="tag.charAt(0)=='!'" icon="minus-circle-outline" size="is-small" class="tagicon"></b-icon>
                  {{removeConditionPrefix(tag)}}
                </b-tooltip>
            </b-tag>
          </template>
        </b-taginput>
      </b-field>

      <b-field label="Tags" label-position="on-border" class="field-extra">
        <b-taginput v-model="tags" autocomplete :data="filteredTags" @typing="getFilteredTags">
          <template slot-scope="props">{{ props.option }}</template>
          <template slot="empty">No matching tags</template>
          <template #selected="props">
            <b-tag v-for="(tag, index) in props.tags"
              :type="tag.charAt(0)=='!' ? 'is-danger': (tag.charAt(0)=='&' ? 'is-success' : '')"
              :key="tag+index" :tabstop="false" closable  @close="tags=tags.filter(e => e !== tag)" @click="toggle3way(tag,index,'tags')"> 
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

      <b-field label="Cuepoint" label-position="on-border" class="field-extra">
        <b-taginput v-model="cuepoint" autocomplete :data="filteredCuepoints" @typing="getFilteredCuepoints">
          <template slot-scope="props">{{ props.option }}</template>
          <template slot="empty">No matching cuepoints</template>
          <template #selected="props">
            <b-tag v-for="(tag, index) in props.tags"
              :type="tag.charAt(0)=='!' ? 'is-danger': (tag.charAt(0)=='&' ? 'is-success' : '')"
              :key="tag+index" :tabstop="false" closable  @close="cuepoint=cuepoint.filter(e => e !== tag)" @click="toggle3way(tag,index,'cuepoints')"> 
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

      <b-tooltip position="is-top" label="Allows searching a variety of attributes such as: scenes in Watchlists, Favourites, Has Video, Scripts or HSP Files, Subscriptions, Ratings, Cuepoint Types, Number of Cast, FOV, Projection, Resolution, Frame Rate and Codecs" multilined :delay="1000" style="width:100%">
        <b-field label="Attributes" label-position="on-border" class="field-extra">        
          <b-taginput v-model="attributes" autocomplete :data="filteredAttributes" @typing="getFilteredAttributes">
            <template slot-scope="props">{{ props.option }}</template>
            <template slot="empty">No matching attributes</template>
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
    </div>
    <div class="is-divider" data-content="Actor Also Known As groups"></div>
    <b-field>
      <b-tooltip position="is-right" label="New Aka Group. Select 2 or more actors in the Cast filter" multilined :delay="200">
        <button class="button is-small is-outlined" @click="createAkaGroup" :disabled="disableNewAkaGroup">
          <b-icon pack="mdi" icon="account-multiple-plus-outline"></b-icon>
        </button>
      </b-tooltip>
      <b-tooltip position="is-right" label="Select the Aka Group to delete in the Cast Filter" multilined :delay="200">
        <button class="button is-small is-outlined" @click="deleteAkaGroup" :disabled="disableDeleteAkaGroup">
          <b-icon pack="mdi" icon="delete-outline"></b-icon>
        </button>
      </b-tooltip>
      <b-tooltip position="is-bottom" label="Add Cast to Aka Group. Select the Aka group and Actors to add in the Cast Filter" multilined :delay="200">
        <button class="button is-small is-outlined" @click="addToAkaGroup" :disabled="disableAddToAkaGroup">
          <b-icon pack="mdi" icon="account-plus-outline"></b-icon>
        </button>
      </b-tooltip>
      <b-tooltip position="is-bottom" label="Remove Cast from Aka Group. Select the Aka group and Actors to remove in the Cast Filter" multilined :delay="200">
        <button class="button is-small is-outlined" @click="removeFromAkaGroup" :disabled="disableRemoveFromAkaGroup">
          <b-icon pack="mdi" icon="account-minus-outline"></b-icon>
        </button>
      </b-tooltip>

    </b-field>
    <div class="is-divider" data-content="Tag Groups"></div>
    <b-field>
      <b-tooltip position="is-right" label="New Tag Group. Select 2 or more tags in the Tag filter" multilined :delay="200">
        <button class="button is-small is-outlined" @click="showGroupTagNameDialog('create')" :disabled="disableNewTagGroup">
          <b-icon pack="mdi" icon="tag-multiple-outline"></b-icon>
        </button>
      </b-tooltip>
      <b-tooltip position="is-right" label="Select the Tag Group to delete in the Tag Filter" multilined :delay="200">
        <button class="button is-small is-outlined" @click="deleteTagGroup" :disabled="disableDeleteRenameTagGroup">
          <b-icon pack="mdi" icon="delete-outline"></b-icon>
        </button>
      </b-tooltip>
      <b-tooltip position="is-bottom" label="Add Tag to Tag Group. Select the Tag  group and Tag to add in the Tag Filter" multilined :delay="200">
        <button class="button is-small is-outlined" @click="addToTagGroup" :disabled="disableAddToTagGroup">
          <b-icon pack="mdi" icon="tag-plus-outline"></b-icon>
        </button>
      </b-tooltip>
      <b-tooltip position="is-bottom" label="Remove Tag from Tag Group. Select the Tag group and Tags to remove in the Tag Filter" multilined :delay="200">
        <button class="button is-small is-outlined" @click="removeFromTagGroup" :disabled="disableRemoveFromTagGroup">
          <b-icon pack="mdi" icon="tag-minus-outline"></b-icon>
        </button>
      </b-tooltip>
      <b-tooltip position="is-bottom" label="Rename Tag Group. Select the Tag group in the Tag Filter" multilined :delay="200">
        <button class="button is-small is-outlined" @click="showGroupTagNameDialog('rename')" :disabled="disableDeleteRenameTagGroup">
          <b-icon pack="mdi" icon="rename-outline"></b-icon>
        </button>
      </b-tooltip>
      <b-tooltip position="is-bottom" label="List Tags in Group" multilined :delay="200">
        <button class="button is-small is-outlined" @click="getTagGroup" :disabled="disableGetTagGroup">
          <b-icon pack="mdi" icon="tag-search-outline"></b-icon>
        </button>
      </b-tooltip>

    <b-modal :active.sync="isGroupTagNameModalActive"
             has-modal-card
             trap-focus
             aria-role="dialog"
             aria-modal>
      <div class="modal-card" style="width: auto">
        <header class="modal-card-head">
          <p class="modal-card-title">Tag Group</p>
        </header>
        <section class="modal-card-body">
          <b-field label="Name">
            <b-input
              type="name"
              v-model="tagGroupName"
              required>
            </b-input>
          </b-field>          
        </section>
        <footer class="modal-card-foot">
          <button class="button is-primary" :disabled="tagGroupName===''" @click="tagGroupModalClicked()">Save
          </button>
        </footer>
      </div>
    </b-modal>

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
    this.$store.dispatch('sceneList/filters')
    this.fetchFilters()
  },
  data () {
    return {
      filteredCast: [],
      filteredSites: [],
      filteredTags: [],
      filteredCuepoints: [],
      filteredAttributes: [],
      isGroupTagNameModalActive: false,
      tagGroupName: '',
      groupNameDialogAction: 'create',
    }
  },
  methods: {
    reloadList () {
      this.$router.push({
        name: 'scenes',
        query: {
          q: this.$store.getters['sceneList/filterQueryParams']
        }
      })
    },
    getFilteredCast (text) {
      this.filteredCast = this.filters.cast.filter(option => (
        option.toString().toLowerCase().indexOf(text.toLowerCase()) >= 0 &&
        !this.cast.some(entry => this.removeConditionPrefix(entry.toString()) === option.toString())
      ))      
    },
    getFilteredSites (text) {
      this.filteredSites = this.filters.sites.filter(option => (
        option.toString().toLowerCase().indexOf(text.toLowerCase()) >= 0 &&
        !this.sites.some(entry => this.removeConditionPrefix(entry.toString()) === option.toString())
      ))
    },
    getFilteredTags (text) {
      this.filteredTags = this.filters.tags.filter(option => (
        option.toString().toLowerCase().indexOf(text.toLowerCase()) >= 0 &&
        !this.tags.some(entry => this.removeConditionPrefix(entry.toString()) === option.toString())
      ))
    },
    getFilteredCuepoints (text) {
      this.filteredCuepoints = this.filters.cuepoints.filter(option => (
        option.toString().toLowerCase().indexOf(text.toLowerCase()) >= 0 &&
        !this.cuepoint.some(entry => this.removeConditionPrefix(entry.toString()) === option.toString())
      ))
    },
    getFilteredAttributes (text) {
      this.filteredAttributes = this.filters.attributes.filter(option => (
        option.toString().toLowerCase().indexOf(text.toLowerCase()) >= 0 &&
        !this.tags.some(entry => this.removeConditionPrefix(entry.toString()) === option.toString())
      ))
    },
    clearReleaseMonth () {
      this.$store.state.sceneList.filters.releaseMonth = ''
      this.reloadList()
    },
    clearVolume () {
      this.$store.state.sceneList.filters.volume = 0
      this.$store.dispatch('sceneList/filters')
      this.reloadList()
    },
    createAkaGroup () {
      this.$store.state.sceneList.isLoading = true
      ky.post('/api/aka/create', {json: {actorList: this.cast}}).json().then(data => {
        this.cast.push(data.akas.aka_actor.name)
        this.$store.dispatch('sceneList/filters')
        this.reloadList()
        if (data.status != '') {
          this.$buefy.toast.open({message: `Warning:  ${data.status}`, type: 'is-warning', duration: 5000})
        }
        this.$store.state.sceneList.isLoading = false
      })
    },
    deleteAkaGroup () {
      this.$store.state.sceneList.isLoading = true
      ky.post('/api/aka/delete', {json: {name: this.cast[0]}}).json().then(data => {
        this.cast = []
        this.$store.dispatch('sceneList/filters')
        this.reloadList()
        this.$store.state.sceneList.isLoading = false
      })       
    },
    addToAkaGroup () {
      this.$store.state.sceneList.isLoading = true
      ky.post('/api/aka/add', {json: {actorList: this.cast}}).json().then(data => {        
        // delete old aka & add new name
        this.cast = this.cast.filter(e => !e.startsWith("aka:")) 
        this.cast.push(data.akas.aka_actor.name) 
        this.$store.dispatch('sceneList/filters')       
        this.reloadList()        
        if (data.status != '') {
          this.$buefy.toast.open({message: `Warning:  ${data.status}`, type: 'is-warning', duration: 5000})
        }
        this.$store.state.sceneList.isLoading = false
      })
      
    },
    removeFromAkaGroup () {
      this.$store.state.sceneList.isLoading = true
      ky.post('/api/aka/remove', {json: {actorList: this.cast}}).json().then(data => {        
        // delete old aka & add new name
        this.cast = this.cast.filter(e => !e.startsWith("aka:")) 
        this.cast.push(data.akas.aka_actor.name)
        this.$store.dispatch('sceneList/filters')
        this.reloadList()
        if (data.status != '') {
          this.$buefy.toast.open({message: `Warning:  ${data.status}`, type: 'is-warning', duration: 5000})
        }
        this.$store.state.sceneList.isLoading = false
      })
    },
    showGroupTagNameDialog (action) {
      this.groupTagName = ''
      this.isGroupTagNameModalActive = true
      this.groupNameDialogAction = action
    },
    tagGroupModalClicked () {
      if (this.groupNameDialogAction == 'create') {
        this.createTagGroup()
      } else {
        this.renameTagGroup()
      }
    },
    createTagGroup () {
      this.isGroupTagNameModalActive = false
      this.$store.state.sceneList.isLoading = true
      ky.post('/api/tag_group/create', {json: {name: this.tagGroupName, tagList: this.tags}}).json().then(data => {
        if (data.tag_group.tag_group_tag.name != "") {
          this.tags.push(data.tag_group.tag_group_tag.name)
        }
        this.$store.dispatch('sceneList/filters')
        this.reloadList()
        if (data.status != '') {
          this.$buefy.toast.open({message: `Warning:  ${data.status}`, type: 'is-warning', duration: 5000})
        }
        this.$store.state.sceneList.isLoading = false
      })
    },
    deleteTagGroup () {
      this.$store.state.sceneList.isLoading = true
      ky.post('/api/tag_group/delete', {json: {name: this.tags[0]}}).json().then(data => {
        this.tags = []
        this.$store.dispatch('sceneList/filters')
        this.reloadList()
        this.$store.state.sceneList.isLoading = false
      })       
    },
    addToTagGroup () {      
      this.$store.state.sceneList.isLoading = true
      ky.post('/api/tag_group/add', {json: {tagList: this.tags}}).json().then(data => {
        this.$store.dispatch('sceneList/filters')       
        this.reloadList()        
        if (data.status != '') {
          this.$buefy.toast.open({message: `Warning:  ${data.status}`, type: 'is-warning', duration: 5000})
        }
        this.$store.state.sceneList.isLoading = false
      })
      
    },
    removeFromTagGroup () {
      this.$store.state.sceneList.isLoading = true
      ky.post('/api/tag_group/remove', {timeout: 60000, json: {tagList: this.tags}}).json().then(data => {        
        this.$store.dispatch('sceneList/filters')
        this.reloadList()
        if (data.status != '') {
          this.$buefy.toast.open({message: `Warning:  ${data.status}`, type: 'is-warning', duration: 5000})
        }
        this.$store.state.sceneList.isLoading = false
      })
    },
    renameTagGroup () {
      this.isGroupTagNameModalActive = false
      this.$store.state.sceneList.isLoading = true
      ky.post('/api/tag_group/rename', {json: {name: this.tagGroupName, tagList: this.tags}}).json().then(data => {
        if (data.status != '') {
          this.$buefy.toast.open({message: `${data.status}`, type: 'is-danger', duration: 5000})
        } else {
          this.reloadList()
          this.tags = []
          this.tags.push(data.tag_group.tag_group_tag.name)
          this.$store.dispatch('sceneList/filters')
        }
        this.$store.state.sceneList.isLoading = false
        })
    },
    getTagGroup () {
      this.$store.state.sceneList.isLoading = true
      let name = ""
      for (var i = 0; i < this.tags.length; i++) {        
        if (this.tags[i].startsWith("tag group:")) {
           name = this.tags[i]
        }
      }

      ky.get('/api/tag_group/' + name, {timeout: 60000}).json().then(data => {        
        if (data.status != '') {
          this.$buefy.toast.open({message: `${data.status}`, type: 'is-danger', duration: 5000})
        } else {
          let newTagList = []
          newTagList.push("tag group:" + data.tag_group.name)
          for (var i = 0; i < data.tag_group.tags.length; i++) {
            newTagList.push(data.tag_group.tags[i].name)
          }
          this.tags = newTagList
          this.$store.dispatch('sceneList/filters')
        }
      })
      this.$store.state.sceneList.isLoading = false
    },
    toggle3way (text, idx, list) {      
      let tags = []
      switch (list) {
        case 'cast':
          tags=this.cast 
          break
        case 'tags':
          tags=this.tags
          break
        case 'cuepoints':
          tags=this.cuepoint
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
        case 'tags':
          this.tags=tags
          break
        case 'cuepoints':
          this.cuepoint=tags
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
        ky.get('/api/scene/filters').json().then(data => {
          this.filteredAttributes=data.attributes          
      })      
    }
  },
  computed: {
    filters () {
      return this.$store.state.sceneList.filterOpts
    },
    lists: {
      get () {
        return this.$store.state.sceneList.filters.lists
      },
      set (value) {
        this.$store.state.sceneList.filters.lists = value
        this.reloadList()
      }
    },
    releaseMonth: {
      get () {
        return this.$store.state.sceneList.filters.releaseMonth
      },
      set (value) {
        this.$store.state.sceneList.filters.releaseMonth = value
        this.reloadList()
      }
    },
    volume: {
      get () {
        return this.$store.state.sceneList.filters.volume
      },
      set (value) {
        this.$store.state.sceneList.filters.volume = value
        this.reloadList()
      }
    },
    cast: {
      get () {
        return this.$store.state.sceneList.filters.cast
      },
      set (value) {
        this.$store.state.sceneList.filters.cast = value
        this.reloadList()
      }
    },
    sites: {
      get () {
        return this.$store.state.sceneList.filters.sites
      },
      set (value) {
        this.$store.state.sceneList.filters.sites = value
        this.reloadList()
      }
    },
    tags: {
      get () {
        return this.$store.state.sceneList.filters.tags
      },
      set (value) {
        this.$store.state.sceneList.filters.tags = value
        this.reloadList()
      }
    },
    cuepoint: {
      get () {
        return this.$store.state.sceneList.filters.cuepoint
      },
      set (value) {
        this.$store.state.sceneList.filters.cuepoint = value
        this.reloadList()
      }
    },
    attributes: {
      get () {
        return this.$store.state.sceneList.filters.attributes
      },
      set (value) {
        this.$store.state.sceneList.filters.attributes = value
        this.reloadList()        
      }
    },
    sort: {
      get () {
        return this.$store.state.sceneList.filters.sort
      },
      set (value) {
        this.$store.state.sceneList.filters.sort = value
        this.reloadList()
      }
    },
    isWatched: {
      get () {
        return this.$store.state.sceneList.filters.isWatched
      },
      set (value) {
        this.$store.state.sceneList.filters.isWatched = value
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
    disableNewTagGroup() {
      let akaTagCnt = 0
      let tagCnt = 0       
 
      for (var i = 0; i < this.tags.length; i++) {        
        if (this.tags[i].startsWith("tag group:")) {
          akaTagCnt++
        } else {
          tagCnt++
        }
      }
      // you can create a new group from a list of tags (more than one)
      return akaTagCnt == 0 && tagCnt > 1 ? false : true
    },
    disableDeleteRenameTagGroup() {
      let tagGroupCnt = 0
      let tagCnt = 0       
 
      for (var i = 0; i < this.tags.length; i++) {        
        if (this.tags[i].startsWith("tag group:")) {
          tagGroupCnt++
        } else {
          tagCnt++
        }
      }

      // you can only delete a group when it is the only thing selected      
      return tagGroupCnt == 1 && tagCnt == 0 > 1 ? false : true
    },
    disableAddToTagGroup() {
      let tagGroupCnt = 0
      let tagCnt = 0       
 
      for (var i = 0; i < this.tags.length; i++) {        
        if (this.tags[i].startsWith("tag group:")) {
          tagGroupCnt++
        } else {
          tagCnt++
        }
      }

      // you can add to a group if you select one group and one or more tags
      return tagGroupCnt == 1 && tagCnt > 0 ? false : true
    },
    disableRemoveFromTagGroup() {
      let tagGroupCnt = 0
      let tagCnt = 0       
 
      for (var i = 0; i < this.tags.length; i++) {        
        if (this.tags[i].startsWith("tag group:")) {
          tagGroupCnt++
        } else {
          tagCnt++
        }
      }

      // you can remove from a group if you select one group and one or more tag
      return tagGroupCnt == 1 && tagCnt > 0 ? false : true

    },
    disableGetTagGroup() {
      let tagGroupCnt = 0
 
      for (var i = 0; i < this.tags.length; i++) {        
        if (this.tags[i].startsWith("tag group:")) {
          tagGroupCnt++
        }
      }

      // you can list a group if you select one 
      return tagGroupCnt == 1 ? false : true

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
</style>
