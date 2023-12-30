<template>
  <div class="modal is-active">
    <GlobalEvents
      :filter="e => !['INPUT', 'TEXTAREA'].includes(e.target.tagName)"
      @keyup.esc="close"
      @keydown.left="handleLeftArrow"
      @keydown.right="handleRightArrow"
      @keydown.o="prevActor"
      @keydown.p="nextActor"
      @keydown.f="$store.commit('actorList/toggleActorList', {actor_id: actor.id, list: 'favourite'})"
      @keydown.exact.w="$store.commit('actorList/toggleActorList', {actor_id: actor.id, list: 'watchlist'})"
      @keydown.e="$store.commit('overlay/editActorDetails', {actor: actor})"
      @keydown.48="setRating(0)"
    />

    <div class="modal-background"></div>

    <div class="modal-card">
      <section class="modal-card-body">
        <div class="columns">

          <div class="column is-half">
            <b-tabs v-model="activeMedia" position="is-centered" :animated="false">
              <b-tab-item :label="$t('Gallery')">
                <b-carousel v-model="carouselSlide" @change="scrollToActiveIndicator" :autoplay="false" :indicator-inside="false">
                  <b-carousel-item v-for="(carousel, i) in images" :key="i">
                    <div class="image is-1by1 is-full"
                         v-bind:style="{backgroundImage: `url(${getImageURL(carousel, '700,fit')})`, backgroundSize: 'contain', backgroundPosition: 'center', backgroundRepeat: 'no-repeat'}"></div>
                  </b-carousel-item>
                  <template slot="indicators" slot-scope="props">
                      <span class="al image" style="width:max-content;">
                        <vue-load-image>
                          <img slot="image" :src="getIndicatorURL(props.i)" style="height:85px;"/>
                          <img slot="preloader" :src="getImageURL('https://i.stack.imgur.com/kOnzy.gif')" style="height:25px;"/>
                          <img slot="error" src="/ui/images/blank_female_profile.png" style="height:85px;"/>
                        </vue-load-image>
                      </span>
                  </template>
                </b-carousel>
                <div class="flexcentre">
                <b-button class="button is-primary is-small" style="display: flex; justify-content: center;" v-on:click="setActorImage()">{{$t('Set Main Image')}}</b-button>
                <b-button v-if="images.length != 0" class="button is-primary is-small" style="display: flex; justify-content: center;margin-left: 1em;" v-on:click="deleteActorImage()">{{$t('Delete Image')}}</b-button>
                </div>
              </b-tab-item>
            </b-tabs>
          </div>

          <div class="column is-half">
            <div class="block-info block">
              <div class="content">
                <h3>
                  <span>
                    {{ actor.name }}
                    <b-tooltip position="is-right" :label="$t('Delete Aka Group')" multilined :delay="200" v-if="actor.name.startsWith('aka:')">
                      <button class="button is-small is-outlined" @click="deleteAkaGroup" >
                        <b-icon pack="mdi" icon="delete-outline"></b-icon>
                      </button>
                    </b-tooltip>
                    <b-tooltip v-if="enableNewAkaGroup()" position="is-right" :label="$t('Create a new Aka Group')" multilined :delay="200">
                      <button class="button is-small is-outlined" @click="createAkaGroup">
                        <b-icon pack="mdi" icon="account-multiple-plus-outline"></b-icon>
                      </button>
                    </b-tooltip>
                  </span>
                  <small v-if="actor.birth_date != '0001-01-01T00:00:00Z'" class="is-pulled-right">
                    {{ format(parseISO(actor.birth_date), "yyyy-MM-dd") }}
                  </small>
                </h3>
                <div class="columns">
                  <div class="column pb-0">
                  </div>
                </div>
                <div class="columns is-vcentered">
                  <div class="column pt-0">
                    <b-field>
                      <strong style="width: 8em;">{{ $t('Your Rating') }}</strong>
                      <star-rating :key="actor.id" v-model="actor.star_rating" :rating="actor.star_rating" @rating-selected="setRating"
                                   :increment="0.5" :star-size="20" :show-rating="true" />
                      <b-tooltip :label="$t('Reset Rating')" position="is-right" :delay="250">
                        <b-icon pack="mdi" icon="autorenew" size="is-small" @click.native="setRating(0)" style="padding-left: 1em;padding-top: .5em;"/>
                      </b-tooltip>
                    </b-field>
                    <b-field>
                      <strong style="width: 8em;">{{ $t('Scene Average') }}</strong>
                    <star-rating :key="actor.id" :rating="Math.round(actor.scene_rating_average * 4) / 4" read-only :increment="0.25" :star-size="20" :show-rating="true" active-color="#7957d5"/>
                    </b-field>

                  </div>
                  <div class="column pt-0">
                    <div class="is-pulled-right">
                      <actor-favourite-button :actor="actor"/>&nbsp;
                      <actor-watchlist-button :actor="actor"/>&nbsp;
                      <actor-edit-button :actor="actor"/>&nbsp;
                    </div>
                  </div>
                </div>
              </div>
            </div>
                        
            <div class="block-opts block">
              <b-tabs v-model="activeTab" :animated="false">
                <b-tab-item :label="$t('Details')">
                  <div class="attribute-container">
                    <b-field v-if="actor.birth_date != '0001-01-01T00:00:00Z'">
                      <strong class="attribute-heading">{{ $t('Age') }}:</strong><span class="attribute-data">{{ calcAge(actor.birth_date) }}</span>
                    </b-field>
                    <b-field v-if="actor.start_year + actor.end_year  != 0">
                      <strong class="attribute-heading">{{ $t('Active') }}:</strong><span class="attribute-data">{{ getYearsActive() }}</span>
                    </b-field>
                    <b-field v-if="actor.nationality">
                      <strong class="attribute-heading">{{ $t('Nationality') }}:</strong>
                      <b-field grouped class="attribute-data">
                        <vue-load-image>
                            <img slot="image" :src="getImageURL(this.getCountryFlag(actor.nationality))" style="height:15px;border: 1px solid black;margin-right:0.5em;"/>
                        </vue-load-image>
                        <small>{{ this.getCountryName(actor.nationality) }}</small>
                      </b-field>
                    </b-field>
                    <b-field v-if="actor.ethnicity">
                      <strong class="attribute-heading">{{ $t('Ethnicity') }}:</strong><small  class="attribute-data">{{ actor.ethnicity }}</small>
                    </b-field>
                    <b-field v-if="actor.hair_color">
                      <strong class="attribute-heading">{{ $t('Hair Color') }}:</strong> <small class="attribute-data">{{ actor.hair_color }}</small>
                    </b-field>
                    <b-field v-if="actor.eye_color">
                      <strong class="attribute-heading">{{ $t('Eye Color') }}:</strong> <small class="attribute-data">{{ actor.eye_color }}</small>
                    </b-field>
                    <b-field v-if="actor.height">
                      <strong class="attribute-heading">{{ $t('Height') }}:</strong> <small class="attribute-data">{{ getHeight(actor.height) }}</small>
                    </b-field>
                    <b-field v-if="actor.weight">
                      <strong class="attribute-heading">{{ $t('Weight') }}:</strong> <small class="attribute-data">{{ getWeight(actor.weight) }}</small>
                    </b-field>
                    <b-field v-if="measurements() != ''">
                      <strong class="attribute-heading">{{ $t('Measurements') }}:</strong> <small class="attribute-data">{{ measurements() }}</small>
                    </b-field>
                    <b-field v-if="actor.breast_type != ''">
                      <strong class="attribute-heading">{{ $t('Breast Type') }}:</strong> <small class="attribute-data">{{ actor.breast_type }}</small>
                    </b-field>
                    <b-field v-if="actor.aliases != '' && actor.aliases != '[]'">
                      <strong class="attribute-heading">{{ $t('Aliases') }}:</strong> <small class="attribute-long-data">{{ joinArray(actor.aliases) }}</small>
                    </b-field>
                    <b-field v-if="actor.tattoos != '' && actor.tattoos != '[]'">
                      <strong class="attribute-heading">{{ $t('Tattoos') }}:</strong> <small class="attribute-long-data">{{ joinArray(actor.tattoos) }}</small>
                    </b-field>
                    <b-field v-if="actor.piercings != '' && actor.piercings != '[]'">
                      <strong class="attribute-heading">{{ $t('Piercings') }}:</strong> <small class="attribute-long-data">{{ joinArray(actor.piercings) }}</small>
                    </b-field>
                  </div>
                  <b-message  v-if="actor.biography != ''">
                      {{ actor.biography }}
                    </b-message>
                </b-tab-item>
                <b-tab-item :label="`Scenes (${actor.scenes.length})`">
                  <div v-show="activeTab == 1" :class="['columns', 'is-multiline', actor.scenes.length > 6 ? 'scroll' : '']">
                    <div :class="['column', 'is-multiline', 'is-one-third']"
                      v-for="(scene, idx) in actor.scenes" :key="idx" class="image-wrapper">
                      <SceneCard :item="scene" :reRead=true />
                    </div>
                  </div>
                </b-tab-item>
                <b-tab-item :label="$t('Akas')" :visible="akas.aka_groups != null || akas.actors != null || akas.possible_akas != null">
                  <div v-show="activeTab == 2">
                    <b-field :label="$t('Aka Groups')" v-if="akas.aka_groups != null &&  akas.aka_groups.length!=0">
                      <div  class="columns is-multiline">
                        <div :class="['column', 'is-multiline', 'is-one-third']"
                          v-for="(akaactor, idx) in akas.aka_groups" :key="idx" class="image-wrapper">
                          <ActorCard :actor="akaactor"/>
                        </div>
                      </div>
                    </b-field>
                    <b-field :label="$t('Other Actors In Groups')" v-if="akas.actors != null &&  akas.actors.length!=0">
                      <div  class="columns is-multiline">
                        <div :class="['column', 'is-multiline', 'is-one-third']"
                          v-for="(akaactor, idx) in akas.actors" :key="idx" class="image-wrapper">
                          <ActorCard :actor="akaactor"/>
                          <b-tooltip position="is-bottom" :label="$t('Remove Cast from Aka Group. Select the Aka group and Actors to remove in the Cast Filter')" multilined :delay="200">
                            <button class="button is-small is-outlined" @click="removeFromAkaGroup(akaactor.name)" v-if="actor.name.startsWith('aka:')">
                              <b-icon pack="mdi" icon="account-minus-outline"></b-icon>
                            </button>
                          </b-tooltip>
                        </div>
                      </div>
                    </b-field>
                    <b-field :label="$t('Possible Matches')" v-if="akas.possible_akas != null &&  akas.possible_akas.length!=0">
                      <div class="columns is-multiline">
                        <div :class="['column', 'is-multiline', 'is-one-third']"
                          v-for="(akaactor, idx) in akas.possible_akas" :key="idx" class="image-wrapper">
                          <ActorCard :actor="akaactor"/>
                          <b-tooltip position="is-bottom" :label="$t('Add Cast to Aka Group. Select the Aka group and Actors to add in the Cast Filter')" multilined :delay="200">
                            <button class="button is-small is-outlined" @click="addToAkaGroup(akaactor.name)" v-if='actor.name.startsWith("aka:")'>
                              <b-icon pack="mdi" icon="account-plus-outline"></b-icon>
                            </button>
                          </b-tooltip>
                        </div>
                      </div>
                    </b-field>
                  </div>
                </b-tab-item>                
                <b-tab-item :visible="colleagues.length != 0" :label="`Colleagues (${colleagues.length})`">
                  <div v-show="activeTab == 3" class="columns is-multiline scroll">
                    <div :class="['column', 'is-multiline', 'is-one-third']"
                      v-for="(colleague, idx) in colleagues" :key="idx" class="image-wrapper">
                      <ActorCard :actor="colleague" :colleague="actor.name" />
                    </div>
                  </div>
                </b-tab-item>
                <b-tab-item :label="`Links (${getActorUrls().length})`" v-show="getActorUrls().length !=0">
                  <div v-show="activeTab == 4">
                    <div >                    
                      <b-field :label="$t('Links')" >
                        <div >                       
                          <div 
                            v-for="(urllink, idx) in getActorUrls()" :key="idx">
                            <a class="tag is-info" :href="urllink.url" target="_blank" rel="noreferrer" style="margin-bottom: .5em;">{{urllink.url}}</a>                            
                          </div>                        
                        </div>
                      </b-field>
                    </div>
                  </div>
                </b-tab-item>
                <b-tab-item  :label="`Scrapers (${extrefs.length})`" v-show="extrefs.length !=0">
                  <div v-show="activeTab == 5">
                    <div >                    
                      <b-field :label="$t('Actor Scrapers')" >
                        <div >                       
                          <div v-for="(extref, idx) in extrefs" :key="idx">
                            <b-field grouped>
                              <a @click="refreshScraper(extref.external_reference.external_url)" :title="'Rescrape Actor Details now'">
                                <b-icon pack="mdi" icon="refresh" size="is-small" style="margin-right: 1em;"/>
                              </a>
                            <a class="tag is-info" :href="extref.external_reference.external_url" target="_blank" rel="noreferrer" style="margin-bottom: .5em;">{{extref.external_source}} - Updated: {{format(parseISO(extref.external_reference.external_date), "yyyy-MM-dd") }}</a>                            
                            </b-field>
                          </div>                        
                        </div>
                      </b-field>
                    </div>
                  </div>
                </b-tab-item>
              </b-tabs>
            </div>

          </div>
        </div>
      </section>
    </div>
    <button class="modal-close is-large" aria-label="close" @click="close()"></button>
    <a class="prev" @click="prevActor"
       title="Keyboard shortcut: O">&#10094;</a>
    <a class="next" @click="nextActor"
       title="Keyboard shortcut: P">&#10095;</a>
  </div>
</template>

<script>
import ky from 'ky'
import videojs from 'video.js'
import 'videojs-vr/dist/videojs-vr.min.js'
import { format, parseISO } from 'date-fns'
import VueLoadImage from 'vue-load-image'
import GlobalEvents from 'vue-global-events'
import StarRating from 'vue-star-rating'
import ActorFavouriteButton from '../../components/ActorFavouriteButton'
import ActorWatchlistButton from '../../components/ActorWatchlistButton'
import ActorEditButton from '../../components/ActorEditButton'
import SceneCard from '../scenes/SceneCard'
import ActorCard from './ActorCard'

export default {
  name: 'ActorDetails',
  components: { VueLoadImage, GlobalEvents, StarRating, ActorWatchlistButton, ActorFavouriteButton, SceneCard, ActorEditButton,  ActorCard },
  data () {
    return {
      index: 1,
      activeTab: 0,
      activeMedia: 0,
      carouselSlide: 0,
      sortMultiple: true,
      countries: [],
      akas: [],
      extrefs: [],
      colleagues: [],
    }
  },
  computed: {
    actor () {      
      const actor = this.$store.state.overlay.actordetails.actor
      ky.get(`/api/actor/akas/${actor.id}`)
      .json()
      .then(list => {          
        this.akas = list
      })
      ky.get(`/api/actor/colleagues/${actor.id}`)
      .json()
      .then(list => {          
        this.colleagues = list
      })
      ky.get(`/api/actor/extrefs/${actor.id}`)
      .json()
      .then(list => {          
        this.extrefs = list          
      })
      return actor
    },
    // Properties for gallery
    images () {
      if (this.actor.image_arr==undefined || this.actor.image_arr=="") {
        return []
      }      
      return JSON.parse(this.actor.image_arr).filter(im => im != "")      
    },
    showEdit () {
      return this.$store.state.overlay.actoredit.show
    },
  },
  mounted () {    
      ky.get('/api/actor/countrylist')
        .json()
        .then(list => {
          this.countries=list
        })
  },
  watch: {
    // when a file is selected, then this will fire the upload process
    activeTab: function (newval, oldval) {      
      }
    },  
    methods: {
    getImageURL (u, size) {
      if (u.startsWith('http') || u.startsWith('https')) {
        return '/img/' + size + '/' + u.replace('://', ':/')
      } else {
        return u
      }
    },
    getIndicatorURL (idx) {      
      if (this.images[idx] !== undefined) {
        return this.getImageURL(this.images[idx], 'x85')
      } else {
        return '/ui/images/blank_female_profile.png'
      }
    },
    close () {      
      this.$store.commit('overlay/hideActorDetails')
    },
    setRating (val) {
      ky.post(`/api/actor/rate/${this.actor.id}`, { json: { rating: val } })
      const updatedActor = Object.assign({}, this.actor)
      updatedActor.star_rating = val
      this.actor.star_rating = val      
      this.$store.commit('actorList/updateActor', updatedActor)
    },
    async nextActor () {      
      const data = this.$store.getters['actorList/nextActor'](this.actor)
      if (data !== null) {
        this.$store.commit('overlay/showActorDetails', { actor: data })
        this.activeMedia = 0
        this.carouselSlide = 0        
      } else {
        // no actor, get the next page (note offset already points to it)
        let newoffset = this.$store.state.actorList.offset
        if (newoffset>this.$store.state.actorList.total)
        {
          // wrap back to the start
          newoffset = 0
        }
        await this.$store.dispatch('actorList/load', { offset: newoffset })
        const data = this.$store.getters['actorList/firstActor'](this.actor)
        if (data !== null) {
          this.$store.commit('overlay/showActorDetails', { actor: data })
          this.activeMedia = 0
          this.carouselSlide = 0
        }
      }
    },
    async prevActor () {
      const data = this.$store.getters['actorList/prevActor'](this.actor)
      if (data !== null) {
        this.$store.commit('overlay/showActorDetails', { actor: data })
        this.activeMedia = 0
        this.carouselSlide = 0        
      } else {
        // no actor, get the previous page
        let newoffset = this.$store.state.actorList.offset - (this.$store.state.actorList.limit * 2)
        if (newoffset < 0) {
          // wrap back to the last actor
          newoffset = Math.floor(this.$store.state.actorList.total / this.$store.state.actorList.limit) * this.$store.state.actorList.limit
        }
        await this.$store.dispatch('actorList/load', { offset: newoffset })
        const data = this.$store.getters['actorList/lastActor'](this.actor)
        if (data !== null) {
          this.$store.commit('overlay/showActorDetails', { actor: data })
          this.activeMedia = 0
          this.carouselSlide = 0
        }
      }
    },    
    handleLeftArrow () {      
        this.carouselSlide = this.carouselSlide - 1
    },
    handleRightArrow () {
        this.carouselSlide = this.carouselSlide + 1
    },
    scrollToActiveIndicator (value) {
      const indicators = document.querySelector('.carousel-indicator')
      const active = indicators.children[value]
      indicators.scrollTo({
        top: 0,
        left: active.offsetLeft + active.offsetWidth / 2 - indicators.offsetWidth / 2,
        behavior: 'smooth'
      })
    },
    calcAge(birthdate){       
      const birthdateObj = new Date(birthdate);
      const now = new Date();
      const diffInMs = now - birthdateObj;
      const msPerYear = 1000 * 60 * 60 * 24 * 365.25; // average milliseconds per year, accounting for leap years
      const age = Math.floor(diffInMs / msPerYear);      
      return age
    },
    getYearsActive(){      
      let active = ""
      if (this.actor.start_year > 0) {
        active = this.actor.start_year 
      }      
      active +=  "-"      
      if (this.actor.end_year > 0) {
        active += this.actor.end_year 
      }
      return active
    },
    measurements(){      
      let metric_measurements=""
      let imperial_measurements=""
      if (this.actor.band_size != 0) {
        metric_measurements=this.actor.band_size
        imperial_measurements=Math.round(this.actor.band_size / 2.54)
      }
      if (this.actor.cup_size != ''){
        metric_measurements +=  this.actor.cup_size        
        imperial_measurements += this.actor.cup_size        
      }
      if (this.actor.waist_size != 0) {
        if (metric_measurements!='') {
          metric_measurements += '-'
          imperial_measurements += '-'
        }
        metric_measurements += this.actor.waist_size
        imperial_measurements += Math.round(this.actor.waist_size  / 2.54)
      }
      if (this.actor.hip_size != 0) {
        if (metric_measurements!='') {
          metric_measurements += '-'
          imperial_measurements += '-'
        }
        metric_measurements += this.actor.hip_size
        imperial_measurements += Math.round(this.actor.hip_size / 2.54)
      } 
      if (metric_measurements==''){
        return ''
      }
      return imperial_measurements + " / " + metric_measurements
    },
    joinArray(jsonArr){
      const arr = JSON.parse(jsonArr);
      return  arr.join(", ");       
    },
    setActorImage (val) {
      ky.post('/api/actor/setimage', {
      json: {
        actor_id: this.actor.id,
        url: this.images[this.carouselSlide]
      }}).json().then(data => {        
        this.$store.state.overlay.actordetails.actor = data
        this.carouselSlide=0
        this.$store.dispatch('actorList/load', { offset: this.$store.state.actorList.offset - this.$store.state.actorList.limit })
      })    
    },
    deleteActorImage (val) {
      ky.delete('/api/actor/delimage', {
      json: {
        actor_id: this.actor.id,
        url: this.images[this.carouselSlide]
      }}).json().then(data => {
        this.$store.state.overlay.actordetails.actor = data
      })    
    },
    getCountryName(countryCode){
      const country = this.countries.find(c => c.code === countryCode)
      if (country == undefined) {
        return countryCode
      }
      return country.name
    },
    getCountryFlag(countryCode){
      const country = this.countries.find(c => c.code === countryCode)
      if (country == undefined) {
        return 'https://flagcdn.com/' + countryCode.toLowerCase() +'.svg'
      }
      return country.flag_url
    },
    getWeight(kg) {
        return kg + " kg - " + Math.round( kg * 2.20462) + " lbs"
    },
    getHeight(cm){
      const totalInches = Math.round(cm / 2.54)
      let feet = Math.floor(totalInches / 12)
      return cm + " cm - " + feet + "' " +  Math.round(totalInches - (feet*12)) + '"'
    },
    getActorUrls() {
      if (this.actor.urls=="")
      {
        return []
      }      
      let array = JSON.parse(this.actor.urls)      
      return array
    },
    createAkaGroup () {
      this.$store.state.actorList.isLoading = true
      let actorlist = [this.actor.name]
      for (let idx = 0; idx < this.akas.possible_akas.length; idx++) {
        actorlist.push(this.akas.possible_akas[idx].name)
      }
      ky.post('/api/aka/create', {json: {actorList: actorlist}}).json().then(data => {
        if (data.status != '') {
          this.$buefy.toast.open({message: `Warning:  ${data.status}`, type: 'is-warning', duration: 5000})
        }
        ky.get('/api/actor/'+this.actor.id).json().then(data => {
          if (data.id != 0){
            this.$store.state.overlay.actordetails.actor = data          
          }          
        })
        this.$store.state.actorList.isLoading = false
        this.$store.dispatch('actorList/load', { offset: this.$store.state.actorList.offset - this.$store.state.actorList.limit })

      })
    },
    deleteAkaGroup () {
      this.$store.state.actorList.isLoading = true
      ky.post('/api/aka/delete', {json: {name: this.actor.name}}).json().then(data => {
        this.$store.state.actorList.isLoading = false
      }).then(data => {
        this.$store.dispatch('actorList/load', { offset: this.$store.state.actorList.offset - this.$store.state.actorList.limit })
        this.close()
      }
      )
    },
    addToAkaGroup (newMember) {
      this.$store.state.actorList.isLoading = true
      ky.post('/api/aka/add', {json: {actorList: [this.actor.name, newMember]}}).json().then(data => {        
        // delete old aka & add new name
        if (data.status != '') {
          this.$buefy.toast.open({message: `Warning:  ${data.status}`, type: 'is-warning', duration: 5000})
        }
        ky.get('/api/actor/'+this.actor.id).json().then(data => {
          if (data.id != 0){
            this.$store.state.overlay.actordetails.actor = data          
          }          
        })
        this.$store.state.actorList.isLoading = false
      })
      
    },
    removeFromAkaGroup (memberToRemove) {
      this.$store.state.actorList.isLoading = true
      ky.post('/api/aka/remove', {json: {actorList: [this.actor.name, memberToRemove]}}).json().then(data => {        
        // delete old aka & add new name
        if (data.status != '') {
          this.$buefy.toast.open({message: `Warning:  ${data.status}`, type: 'is-warning', duration: 5000})
        }
        ky.get('/api/actor/'+this.actor.id).json().then(data => {          
          if (data.id != 0){
            this.$store.state.overlay.actordetails.actor = data          
          }          
        })
        this.$store.state.actorList.isLoading = false
      })
    },
    enableNewAkaGroup () {
      if (this.actor.name.startsWith("aka:")){
        return false
      }
      if (this.akas.aka_groups != null)
      {
        return false
      }
      if (this.akas.possible_akas == null)
      {
        return false
      }
      for (let idx = 0; idx < this.akas.possible_akas.length; idx++) {
        if (this.akas.possible_akas[idx].name.startsWith("aka:")) {
          return false
        }
      }      
      return true
    },
    refreshScraper(url){
      if (url.includes('stashdb')) {
        this.$store.state.actorList.isLoading = true
        const lastSlashIndex = url.lastIndexOf('/');
        ky.get('/api/extref/stashdb/refresh_performer/'+url.substring(lastSlashIndex + 1)).then(data => {
          ky.get('/api/actor/'+this.actor.id).json().then(data => {          
            if (data.id != 0){
              this.$store.state.overlay.actordetails.actor = data
              this.$store.state.actorList.isLoading = false
              this.$store.dispatch('actorList/load', { offset: this.$store.state.actorList.offset - this.$store.state.actorList.limit })
            }
          })
        })
      } else {
        this.$store.state.actorList.isLoading = true
        ky.post('/api/extref/generic/scrape_single', { json: {id: this.actor.id,url: url}})
          .then(data => {
            ky.get('/api/actor/'+this.actor.id).json().then(data => {
              if (data.id != 0){
                this.$store.state.overlay.actordetails.actor = data
                this.$store.state.actorList.isLoading = false
                this.$store.dispatch('actorList/load', { offset: this.$store.state.actorList.offset - this.$store.state.actorList.limit })
              }
            })
          })
      }
      this.$store.state.actorList.isLoading = false
    },
    format,
    parseISO
  }
}
</script>

<style lang="less" scoped>
.bbox {
  flex: 1 0 calc(25%);
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  padding: 0;
  line-height: 0;
}

.is-1by1 {
  padding-top: calc(100% - 40px - 1em) !important;
}

.modal-card {
  width: 85%;
}

.missing {
  opacity: 0.6;
}

.block-tab-content {
  flex: 1 1 auto;
}

.block-info {
}

.block-tags {
  max-height: 200px;
  overflow: scroll;
  scrollbar-width: none;
}

.block-tags::-webkit-scrollbar {
  display: none;
}

.block-opts {
}

.vue-star-rating {
    line-height: 0;
}

.scene-id {
  position: absolute;
  right:10px;
  bottom: 5px;
  font-size: 11px;
  color: #b0b0b0;
}

.prev, .next {
  cursor: pointer;
  position: absolute;
  top: 50%;
  width: auto;
  padding: 16px;
  margin-top: -50px;
  color: white;
  font-weight: bold;
  font-size: 24px;
  border-radius: 0 3px 3px 0;
  user-select: none;
  -webkit-user-select: none;
}

.next {
  right: 0;
  border-radius: 3px 0 0 3px;
}

.prev {
  left: 0;
  border-radius: 3px 0 0 3px;
}

span.is-active img {
  border: 2px;
}

.pathDetails {
  color: #b0b0b0;
}

.heatmapFunscript {
  width: 100%;
  padding: 0;
  margin-top: 0.5em;
}

.heatmapFunscript img {
  border: 1px #888 solid;
  width: 100%;
  height: 20px;
  margin: 0;
  padding: 0;
}
.videosize {
  color: rgb(60, 60, 60);
  font-weight: 550;
}

:deep(.carousel .carousel-indicator) {
  justify-content: flex-start;
  width: 100%;
  max-width: min-content;
  margin-left: auto;
  margin-right: auto;
  overflow: auto;
}
:deep(.carousel .carousel-indicator .indicator-item:not(.is-active)) {
  opacity: 0.5;
}
.is-divider {
  margin: .8rem 0;
}
.image-row {
  display: flex;  
}
.image-wrapper {
  position: relative;
}
.thumbnail {
  height: 100px;
  margin-right: .5em;
  object-fit: cover;
}
.tooltip {
  position: absolute;
  z-index: 1;
  top: 50px;
  right: 100%;
  width: 400px;
  height: 400px;
  background-color: white;
  box-shadow: 0 2px 5px rgba(0, 0, 0, 0.3);
  display: flex;
  justify-content: center;
  align-items: center;
  padding: 10px;
  transform: translateX(10px);
}
.tooltip img {
  max-width: 100%;
  max-height: 100%;
}
div.scroll {
  height: 1000px;
  overflow-x: hidden;
  overflow-y: auto;
  text-align: center;
}
.attribute-container {  
  display: flex; 
  flex-wrap: wrap;
}
.attribute-heading {  
  width: 120px; 
}
.attribute-data {  
  width: 200px;  
}
.attribute-long-data {  
  min-width: 320px;  
}
.flexcentre {
  display: flex;
  justify-content: center;
}

</style>