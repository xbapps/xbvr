<template>
  <div class="modal is-active">
    <GlobalEvents
      :filter="e => !['INPUT', 'TEXTAREA'].includes(e.target.tagName)"
      @keyup.esc="close"
      @keyup.s="save"/>

    <div class="modal-background"></div>

    <div class="modal-card">
      <header class="modal-card-head">
        <p class="modal-card-title">{{ $t('Edit actor details') }} - {{ actor.name }}</p>
        <button class="delete" @click="close" aria-label="close"></button>
      </header>

      <section class="modal-card-body">
        <b-tabs position="is-centered" :animated="false">

          <b-tab-item :label="$t('Information')">
            <b-field grouped group-multiline style="margin-bottom: 2em;">
              <b-field :label="$t('Nationality')" label-position="on-border" class="field-extra">
                <b-taginput v-model="countries" autocomplete :data="filteredCountries" @typing="getFilteredCountries" maxtags="1" :open-on-focus=true :has-counter="false">
                  <template slot-scope="props">{{ props.option }}</template>
                  <template slot="empty">{{ $t('No matching country') }}</template>
                  <template #selected="props">
                      <b-tag v-for="(tag, index) in props.tags"                
                        :key="tag+index" :tabstop="false" closable @close="countries=countries.filter(e => e !== tag)" >
                          {{tag}}
                      </b-tag>
                  </template>
                </b-taginput>
              </b-field>              
              <b-field :label="$t('Ethnicity')" label-position="on-border">
                <b-input type="text" v-model="actor.ethnicity" @blur="blur('ethnicity')"/>
              </b-field>
               <b-datepicker v-model="birthdate" icon="calendar-today" @blur="blur('birth_date')">
                 <b-button :label="$t('Clear')" type="is-danger" icon-left="close" outlined @click="birthdate = null" />
               </b-datepicker>
            </b-field>
            <b-field grouped group-multiline style="margin-bottom: 2em;">
              <b-field :label="$t('Eye Color')" label-position="on-border">
                <b-input type="text" v-model="actor.eye_color" @blur="blur('eye_color')"/>
              </b-field>
              <b-field :label="$t('Hair Color')" label-position="on-border">
                <b-input type="text" v-model="actor.hair_color" @blur="blur('hair_color')"/>
              </b-field>
            </b-field>
            <b-field grouped group-multiline style="margin-bottom: 2em;">
              <b-field v-if="useImperialEntry" :label="$t('Weight in lbs')" label-position="on-border">
                <b-input type="number" v-model.number="actor.lbs" :placeholder="$t('Enter Weight in lbs')"  @blur="blur('weight')"/>                 
              </b-field>
              <b-field v-if="!useImperialEntry" :label="$t('Weight')" label-position="on-border">
                <b-input type="number" v-model.number="actor.weight" :placeholder="$t('Enter Weight in kg')"  @blur="blur('weight')"/>                 
              </b-field>
              <b-field>
              <b-field v-if="useImperialEntry" :label="$t('Height feet/inches')" label-position="on-border">
                <b-input type="number" v-model.number="actor.feet" min="0" max="10" placeholder="Height in feet" @blur="blur('height')" style="width: 5em;"/>
                <b-input type="number" v-model.number="actor.inches" min="0" max="12" placeholder="Height in inches" @blur="blur('height')" style="width: 5em;"/>
              </b-field>
              </b-field>
              <b-field v-if="!useImperialEntry" :label="$t('Height')" label-position="on-border">
                <b-input type="number" v-model.number="actor.height"  placeholder="Height in cm" @blur="blur('height')"/>
              </b-field>
            </b-field>
            <b-field grouped group-multiline style="margin-bottom: 2em;">
              <b-field :label="$t('Measurements')" label-position="on-border">
                <b-input type="text" v-model="actor.measurements" placeholder="eg 36C-24-36" pattern="(^(\d{2})?([A-Za-z]{0,2})-(\d{2})?-(\d{2}$)?)|^[A-Z]{0,2}$" validation-message="use the format 99A-99-99"
                  @blur="blur('measurements')"/>
              </b-field>
              <b-field :label="$t('Breast Type')" label-position="on-border">
                <b-input type="text" v-model="actor.breast_type" placeholder="eg Fake, Natural" @blur="blur('breast_type')"/>
              </b-field>
            </b-field>
            <b-field grouped group-multiline style="margin-bottom: 2em;">
              <b-field :label="$t('Active From')" label-position="on-border">
                <b-input type="number" v-model.number="actor.start_year" :max="new Date().getFullYear()" pattern="^[1-2]\d{1,3}$|^0$|^$"  validation-message="Up to the current year" @blur="blur('start_year')"/>
              </b-field>
              <b-field :label="$t('Active To')" label-position="on-border">
                <b-input type="number" v-model.number="actor.end_year" :max="new Date().getFullYear()" pattern="^[1-2]\d{1,3}$|^0$|^$"  validation-message="Up to the current year" @blur="blur('end_year')"/>
              </b-field>
            </b-field>
            <b-field :label="$t('Biography')" label-position="on-border">
              <b-input type="textarea" v-model="actor.biography" @blur="blur('biography')"/>
            </b-field>
          </b-tab-item>

          <b-tab-item :label="$t('Aliases')">
            <ListEditor :list="this.actor.aliasArray" type="aliases" :blurFn="() => blur('aliases')"/>
          </b-tab-item>
          <b-tab-item :label="$t('Tattoos')">
            <ListEditor :list="this.actor.tattooArray" type="tattoos" :blurFn="() => blur('tattoos')"/>
          </b-tab-item>
          <b-tab-item :label="$t('Piercings')">
            <ListEditor :list="this.actor.piercingArray" type="piercings" :blurFn="() => blur('piercings')"/>
          </b-tab-item>

          <b-tab-item :label="$t('Links')">
            <ListEditor :list="this.actor.urlArray" type="urls" :blurFn="() => blur('urls')" :showUrl="true"/>
          </b-tab-item>

          <b-tab-item :label="$t('Images')">
            <ListEditor :list="this.actor.imageArray" type="image_arr" :blurFn="() => blur('image_arr')" :showUrl="true"/>
          </b-tab-item>
          <b-tab-item :label="$t('Actor Scraper')">
            <ListEditor :list="this.extrefsArray" type="extrefs_arr" :blurFn="() => extrefBlur()" :showUrl="true"/>
          </b-tab-item>
        </b-tabs>

      </section>

      <footer class="modal-card-foot">
        <b-field>
          <b-button type="is-primary" @click="save">{{ $t('Save Details') }}</b-button>
          <b-button v-if="actor.scenes.length == 0 && !actor.name.startsWith('aka:')" type="is-danger" outlined @click="deleteactor">{{ $t('Delete Actor') }}</b-button>
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
  name: 'EditActor',
  components: { ListEditor, GlobalEvents },
  data () {
    const actor = Object.assign({}, this.$store.state.overlay.actoredit.actor)
    let images;
    try {
      images = JSON.parse(actor.image_arr)
    } catch {
      images = []
    }    
    actor.imageArray = images.map(i => i)    
    try {
      actor.aliasArray = JSON.parse(actor.aliases)
    } catch {
      actor.aliasArray = []
    }
    try {
      actor.tattooArray = JSON.parse(actor.tattoos)
    } catch {
      actor.tattooArray = []
    }
    try {
      actor.piercingArray = JSON.parse(actor.piercings)
    } catch {
      actor.piercingArray = []
    }
    actor.measurements = Math.round(actor.band_size / 2.54) + actor.cup_size + '-' + Math.round(actor.waist_size / 2.54) + '-' + Math.round(actor.hip_size / 2.54)
    this.convertCountryCodeToName()
    let urls;
    try {
      urls = JSON.parse(actor.urls)
    } catch {
      urls = []
    }    
    actor.urlArray = urls.map(i => i.url)    

    const totalInches = Math.round(actor.height / 2.54)
    const  feet = Math.floor(totalInches / 12)
    const inches =  Math.round(totalInches - (feet*12))      
    const lbs = Math.round(actor.weight * 220462 / 100000);
    actor.feet = feet
    actor.inches = inches
    actor.lbs = lbs

    return {
      actor,
      // A shallow copy won't work, need a deep copy
      source: JSON.parse(JSON.stringify(actor)),
      changesMade: false,
      extrefsChangesMade: false,
      countryList: [],
      countries: [],
      selectedCountry: '',
      filteredCountries: [],
      extrefsArray: [],
      extrefsSource: '',
    }
  },
  computed: {
    birthdate: {
      get () {        
        if (this.actor.birth_date=='0001-01-01T00:00:00Z') {
          return new Date()
        }
        return new Date(this.actor.birth_date)
      },
      set (value) {        
        if (value==null){
          this.actor.birth_date=null
        }else{
        // remove the time offset, or toISOString may result in a different date
        let adjustedDate = new Date(value.getTime() - (value.getTimezoneOffset() * 60000))
        this.actor.birth_date = adjustedDate.toISOString().split('.')[0] + 'Z'        
        }
      }
    },
    useImperialEntry () {
      return this.$store.state.optionsAdvanced.advanced.useImperialEntry
    },
  },
  mounted () {
    ky.get('/api/actor/countrylist')
    .json()
    .then(list => {
      this.countryList = list
      this.convertCountryCodeToName()
    })  

  ky.get(`/api/actor/extrefs/${this.actor.id}`)
    .json()
    .then(list => {
      this.extrefsArray = []
      list.forEach(extref => {
        this.extrefsArray.push(extref.external_reference.external_url)
      }      
      )
      this.extrefsSource = JSON.parse(JSON.stringify(this.extrefsArray))
      this.extrefsChangesMade=false
    })
  },
  methods: {
    close () {
      if (this.changesMade || this.extrefsChangesMade) {
        this.$buefy.dialog.confirm({
          title: 'Close without saving',
          message: 'Are you sure you want to close before saving your changes?',
          confirmText: 'Close',
          type: 'is-warning',
          hasIcon: true,
          onConfirm: () => this.$store.commit('overlay/hideActorEditDetails')
        })
        return
      }
      this.$store.commit('overlay/hideActorEditDetails')
    },
    async save () {
      this.$store.state.actorList.isLoading = true
      if (this.useImperialEntry) {
        this.actor.height = Math.round(((this.actor.feet * 12) + this.actor.inches) * 2.54)
        this.actor.weight = Math.round(this.actor.lbs * 453592 / 1000000);
      }
      this.actor.aliases = JSON.stringify(this.actor.aliasArray)      
      this.actor.tattoos = JSON.stringify(this.actor.tattooArray)         
      this.actor.piercings = JSON.stringify(this.actor.piercingArray)
      if (this.countries.length==0){
        this.actor.nationality=""
      } else {
        this.actor.nationality=this.countries[0]
      }

      let  dataArray = []
      if (this.actor.urls != "") {
        const existingurls = JSON.parse(this.actor.urls)      
        this.actor.urlArray.forEach(url => {        
          let t = ''
          existingurls.forEach(u => {
            if (u.url==url) {
              t=u.type
            }
          })
          dataArray.push({
            url,
            type: t
          })
        })
      }
      this.actor.height = parseInt(this.actor.height)
      this.actor.weight = parseInt(this.actor.weight)
      this.actor.start_year = parseInt(this.actor.start_year)
      this.actor.end_year = parseInt(this.actor.end_year)

      this.actor.urls = JSON.stringify(dataArray)

      this.actor.image_arr = JSON.stringify(this.actor.imageArray)  

      await ky.post(`/api/actor/edit/${this.actor.id}`, { json: { ...this.actor } })
      await ky.post(`/api/actor/edit_extrefs/${this.actor.id}`, { json: this.extrefsArray  })
      await ky.get('/api/actor/'+this.actor.id).json().then(data => {
        if (data.id != 0){
          this.$store.state.overlay.actordetails.actor = data          
        }          
      })

      this.$store.dispatch('actorList/load', { offset: this.$store.state.actorList.offset - this.$store.state.actorList.limit })
      this.changesMade = false
      this.extrefsChangesMade = false
      this.$store.state.actorList.isLoading = false
      this.close()
    },
    deleteactor () {
      this.$buefy.dialog.confirm({
        title: 'Delete actor',
        message: `Do you really want to delete <strong>${this.actor.name}</strong>`,
        type: 'is-info is-wide',
        hasIcon: true,
        id: 'heh',
        onConfirm: () => {
          ky.delete(`/api/actor/delete/${this.actor.id}`).json().then(data => {
            this.$store.dispatch('actorList/load', { offset: this.$store.state.actorList.offset - this.$store.state.actorList.limit })
            this.$store.commit('overlay/hideActorEditDetails')
            this.$store.commit('overlay/hideActorDetails')
          })
        }
      })
    },
    blur (field) {
      if (this.changesMade) return // Changes have already been made. No point to check any further   
      if (['image_arr', 'tattoos', 'piercings', 'aliases', 'urls'].includes(field)) {
        if (this.actor[field].length !== this.source[field].length) {
          this.changesMade = true
        } else {
          // change to actor and use foreah 
          for (let i = 0; i < this.actor[field].length; i++) {
            if (this.actor[field][i] !== this.source[field][i]) {
              this.changesMade = true
              break
            }
          }
        }
      } else if (this.actor[field] !== this.source[field]) {       
        this.changesMade = true
      }      
    },
    extrefBlur () {      
      if (this.extrefsChangesMade) return // Changes have already been made. No point to check any further         
      if (this.extrefsArray.length !== this.extrefsSource.length) {
        this.extrefsChangesMade = true
      } else {
        // change to actor and use foreah 
        for (let i = 0; i < this.extrefsArray.length; i++) {
          if (this.extrefsArray[i] !== this.extrefsSource[i]) {
            this.extrefsChangesMade = true
            break
          }
        }
      }      
    },
    getFilteredCountries (text) {
      const filtered = this.countryList.filter(option => (
        option.name.toString().toLowerCase().indexOf(text.toLowerCase()) >= 0        
      ))
      this.filteredCountries=[]
      filtered.forEach(item => this.filteredCountries.push(item.name))      
    },
    getYear (text) {
      if (text==0) {
        return ""
      }
      return year
    },
    convertCountryCodeToName() {      
      if (this.countryList != undefined && this.actor != undefined && this.actor.nationality.length == 2) {
        this.countryList.forEach(country => {
          if (country.code == this.actor.nationality) {
            this.actor.nationality=country.name
          }
        })
      }

      if (this.actor != undefined){      
        this.countries = [this.actor.nationality]
      }      
    },
  },
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
