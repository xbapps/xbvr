<template>
  <div>
    <GlobalEvents
      :filter="e => !['INPUT', 'TEXTAREA'].includes(e.target.tagName)"
      @keydown.left="prevpage"
      @keydown.right="nextpage"
      @keydown.o="prevpage"
      @keydown.p="nextpage"
    />
    <b-loading :is-full-page="true" :active.sync="isLoading"></b-loading>

    <div class="columns is-multiline is-full">
      <div class="column">
        <strong>{{total}} results</strong>
      </div>
      <div class="column">
        <b-tooltip :label="$t('Press o/left arrow to page back, p/right arrow to page forward')" :delay="500" position="is-top">
          <b-pagination
              :total="total"
              v-model="current"
              range-before=1
              range-after=3    
              size="is-small"                                           
              :per-page="limit"
              aria-next-label="Next page"
              aria-previous-label="Previous page"
              aria-page-label="Page"
              aria-current-label="Current page"
              :page-input=true
              @change="pageChanged"
              debounce-page-input="250"
              >
          </b-pagination>
        </b-tooltip>
        <span v-show="show_actor_id==='never show, just need the computed show_actor_id to trigger '">{{show_actor_id}}</span>
      </div>
      <div class="column">
        <div class="is-pulled-right">
          <b-field>
            <span class="list-header-label">{{$t('Card size')}}</span>
            <b-radio-button v-model="cardSize" native-value="1" size="is-small">
              S
            </b-radio-button>
            <b-radio-button v-model="cardSize" native-value="2" size="is-small">
              M
            </b-radio-button>
            <b-radio-button v-model="cardSize" native-value="3" size="is-small">
              L
            </b-radio-button>
          </b-field>
        </div>
      </div>
    </div>
        <div class="columns is-gapless is-centered" v-if="hideLetters">
          <b-radio-button v-model="jumpTo" native-value="" size="is-small"></b-radio-button>
          <b-radio-button v-model="jumpTo" native-value="A" size="is-small">A</b-radio-button>
          <b-radio-button v-model="jumpTo" native-value="B" size="is-small">B</b-radio-button>
          <b-radio-button v-model="jumpTo" native-value="C" size="is-small">C</b-radio-button>
          <b-radio-button v-model="jumpTo" native-value="D" size="is-small">D</b-radio-button>
          <b-radio-button v-model="jumpTo" native-value="E" size="is-small">E</b-radio-button>
          <b-radio-button v-model="jumpTo" native-value="F" size="is-small">F</b-radio-button>
          <b-radio-button v-model="jumpTo" native-value="G" size="is-small">G</b-radio-button>
          <b-radio-button v-model="jumpTo" native-value="H" size="is-small">H</b-radio-button>
          <b-radio-button v-model="jumpTo" native-value="I" size="is-small">I</b-radio-button>
          <b-radio-button v-model="jumpTo" native-value="J" size="is-small">J</b-radio-button>
          <b-radio-button v-model="jumpTo" native-value="K" size="is-small">K</b-radio-button>
          <b-radio-button v-model="jumpTo" native-value="L" size="is-small">L</b-radio-button>
          <b-radio-button v-model="jumpTo" native-value="M" size="is-small">M</b-radio-button>
          <b-radio-button v-model="jumpTo" native-value="N" size="is-small">N</b-radio-button>
          <b-radio-button v-model="jumpTo" native-value="O" size="is-small">O</b-radio-button>
          <b-radio-button v-model="jumpTo" native-value="P" size="is-small">P</b-radio-button>
          <b-radio-button v-model="jumpTo" native-value="Q" size="is-small">Q/R</b-radio-button>          
          <b-radio-button v-model="jumpTo" native-value="S" size="is-small">S</b-radio-button>
          <b-radio-button v-model="jumpTo" native-value="T" size="is-small">T</b-radio-button>
          <b-radio-button v-model="jumpTo" native-value="U" size="is-small">U/V</b-radio-button>          
          <b-radio-button v-model="jumpTo" native-value="W" size="is-small">W/X/Y/Z</b-radio-button>
        </div>

    <div class="is-clearfix"></div>

    <div class="columns is-multiline">
      <div :class="['column', 'is-multiline', cardSizeClass]"
           v-for="actor in actors" :key="actor.id">
        <ActorCard :actor="actor"/>
      </div>
    </div>
      <div class="columns is-gapless is-centered" v-if="hideLetters">
        <b-radio-button v-model="jumpTo" native-value="" size="is-small"></b-radio-button>
        <b-radio-button v-model="jumpTo" native-value="A" size="is-small">A</b-radio-button>
        <b-radio-button v-model="jumpTo" native-value="B" size="is-small">B</b-radio-button>
        <b-radio-button v-model="jumpTo" native-value="C" size="is-small">C</b-radio-button>
        <b-radio-button v-model="jumpTo" native-value="D" size="is-small">D</b-radio-button>
        <b-radio-button v-model="jumpTo" native-value="E" size="is-small">E</b-radio-button>
        <b-radio-button v-model="jumpTo" native-value="F" size="is-small">F</b-radio-button>
        <b-radio-button v-model="jumpTo" native-value="G" size="is-small">G</b-radio-button>
        <b-radio-button v-model="jumpTo" native-value="H" size="is-small">H</b-radio-button>
        <b-radio-button v-model="jumpTo" native-value="I" size="is-small">I</b-radio-button>
        <b-radio-button v-model="jumpTo" native-value="J" size="is-small">J</b-radio-button>
        <b-radio-button v-model="jumpTo" native-value="K" size="is-small">K</b-radio-button>
        <b-radio-button v-model="jumpTo" native-value="L" size="is-small">L</b-radio-button>
        <b-radio-button v-model="jumpTo" native-value="M" size="is-small">M</b-radio-button>
        <b-radio-button v-model="jumpTo" native-value="N" size="is-small">N</b-radio-button>
        <b-radio-button v-model="jumpTo" native-value="O" size="is-small">O</b-radio-button>
        <b-radio-button v-model="jumpTo" native-value="P" size="is-small">P</b-radio-button>
        <b-radio-button v-model="jumpTo" native-value="Q" size="is-small">Q/R</b-radio-button>        
        <b-radio-button v-model="jumpTo" native-value="S" size="is-small">S</b-radio-button>
        <b-radio-button v-model="jumpTo" native-value="T" size="is-small">T</b-radio-button>
        <b-radio-button v-model="jumpTo" native-value="U" size="is-small">U/V</b-radio-button>        
        <b-radio-button v-model="jumpTo" native-value="W" size="is-small">W/X/Y/Z</b-radio-button>
      </div>
      <div class="columns is-gapless is-centered">          
        <b-tooltip :label="$t('Press k to page back, l to page forward')" :delay="500" position="is-top">
          <b-pagination
            :total="total"
            v-model="current"
            range-before=2
            range-after=3 
            size="is-small"                                              
            :per-page="limit"
            aria-next-label="Next page"
            aria-previous-label="Previous page"
            aria-page-label="Page"
            aria-current-label="Current page"
            :page-input=true
            @change="pageChanged"
            debounce-page-input="250"
            >
        </b-pagination>
      </b-tooltip>
      </div>
  </div>
</template>

<script>
import ActorCard from './ActorCard'
import ky from 'ky'
import GlobalEvents from 'vue-global-events'

export default {
  name: 'List',
  components: { ActorCard, GlobalEvents },
  data () {
    return {      
      current: 1,      
    }
  },
  computed: {
    cardSize: {
      get () {
        return this.$store.state.actorList.filters.cardSize
      },
      set (value) {
        this.$store.state.actorList.filters.cardSize = value
        switch (value){
          case "1":
            this.limit=18
            break
          case "2":
            this.limit=10
            break
          case "3":
            this.limit=8
            break
            }            
        }      
    },
    limit: {
      get(){
        return this.$store.state.actorList.limit
      },
      set(newLimit){
        // find the position of the first actor
        let currentOffset = this.$store.state.actorList.offset - this.$store.state.actorList.limit + 1
        // what is the new page number, based on the new limit
        this.current = Math.floor(currentOffset / newLimit) + 1
        if (this.current<1)
          this.current=1
        this.$store.state.actorList.limit = newLimit
        // what is the the first actor based on the new page size
        this.$store.state.actorList.offset = (this.current -1) * this.$store.state.actorList.limit          
        this.$store.dispatch('actorList/load', { offset: this.$store.state.actorList.offset })
      }
    },
    jumpTo: {
      get () {
        return this.$store.state.actorList.filters.jumpTo
      },
      set (value) {
        this.$store.state.actorList.filters.jumpTo = value
        this.reloadList()
      }
    },
    cardSizeClass () {
      switch (this.$store.state.actorList.filters.cardSize) {
        case '1':
          return 'is-2'
        case '2':
          return 'is-one-fifth'
        case '3':
          return 'is-one-quarter'
        default:
          return 'is-2'
      }
    },
    isLoading () {
      this.current = this.$store.state.actorList.offset / this.$store.state.actorList.limit
      return this.$store.state.actorList.isLoading
    },
    actors () {
      return this.$store.state.actorList.actors
    },
    total () {
      return this.$store.state.actorList.total
    },
    show_actor_id() {
      if (this.$store.state.actorList.show_actor_id != undefined && this.$store.state.actorList.show_actor_id !='')
      {
        ky.get('/api/actor/'+this.$store.state.actorList.show_actor_id).json().then(data => {
          if (data.id != 0){
            this.$store.commit('overlay/showActorDetails', { actor: data })
          }          
        })
        this.$store.state.actorList.show_actor_id = ''
      }
      
      return this.$store.state.actorList.show_actor_id
    },
    hideLetters: {
      get () {        
        switch (this.$store.state.actorList.filters.sort) {
          case "":
            return true
          case "name_asc":
            return true
          case "name_desc":
            return true
        }
        return false
        },
    },
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
    async pageChanged () {      
      this.$store.state.actorList.offset = (this.current -1) * this.$store.state.actorList.limit
      this.$store.dispatch('actorList/load', { offset: this.$store.state.actorList.offset })
    },
    nextpage () {
      if (this.$store.state.overlay.actordetails.show){
        return 
      }
      if (this.$store.state.overlay.details.show){
        return 
      }
      if (this.current * this.limit >= this.total) {
        this.current = 1
      } else {
        this.current += 1
      }      
      this.pageChanged()
    },
    prevpage () {      
      if (this.$store.state.overlay.actordetails.show){
        return 
      }
      if (this.$store.state.overlay.details.show){
        return 
      }
      if (this.current > 1) {
        this.current -= 1
      } else {
        this.current = Math.floor(this.total / this.limit) + 1        
      }      
      this.pageChanged()
    },
  }
}
</script>

<style scoped>
  .list-header-label {
    padding-right: 1em;
  }
</style>
