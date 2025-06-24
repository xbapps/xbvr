<template>
  <div>
    <b-loading :is-full-page="true" :active.sync="isLoading"></b-loading>

    <div class="columns is-multiline is-full">
      <div class="column">
        <strong>{{total}} results</strong>
      </div>
      <div class="column">
        <div class="columns is-gapless">
          <b-radio-button v-model="dlState" native-value="any" size="is-small">
            {{$t("Any")}} ({{counts.any}})
          </b-radio-button>
          <b-radio-button v-model="dlState" native-value="available" size="is-small">
            {{$t("Available right now")}} ({{counts.available}})
          </b-radio-button>
          <b-radio-button v-model="dlState" native-value="downloaded" size="is-small">
            {{$t("Downloaded")}} ({{counts.downloaded}})
          </b-radio-button>
          <b-radio-button v-model="dlState" native-value="missing" size="is-small">
            {{$t("Not downloaded")}} ({{counts.not_downloaded}})
          </b-radio-button>
          <b-radio-button v-model="dlState" native-value="hidden" size="is-small">
            {{$t("Hidden")}} ({{counts.hidden}})
          </b-radio-button>
        </div>
        <span v-show="show_scene_id==='never show, just need the computed show_scene_id to trigger '">{{show_scene_id}}</span>
      </div>
      <div class="column">
        <div class="is-pulled-right">
          <b-field>
            <span class="list-header-label">{{$t('Card size')}}</span>
            <b-radio-button v-model="cardSize" native-value="1" size="is-small">
              XS
            </b-radio-button>
            <b-radio-button v-model="cardSize" native-value="2" size="is-small">
              S
            </b-radio-button>
            <b-radio-button v-model="cardSize" native-value="3" size="is-small">
              M
            </b-radio-button>
            <b-radio-button v-model="cardSize" native-value="4" size="is-small">
              L
            </b-radio-button>
          </b-field>
        </div>
      </div>
    </div>

    <div class="is-clearfix"></div>

    <div class="columns is-multiline">
      <div :class="['column', 'is-multiline', cardSizeClass]"
           v-for="item in items" :key="item.id">
        <SceneCard :item="item"/>
      </div>
    </div>

    <!-- InfiniteScroll trigger element -->
    <div ref="infiniteScrollTrigger" class="infiniteScroll-trigger"></div>

    <div class="column is-full" v-if="items.length < total && !infiniteScrollEnabled">
      <a class="button is-fullwidth" v-on:click="loadMore()">{{$t('Load more')}}</a>
    </div>

  </div>
</template>

<script>
import SceneCard from './SceneCard'
import ky from 'ky'

export default {
  name: 'List',
  components: { SceneCard },
  data() {
    return {
      infiniteScrollObserver: null,
      loadMoreDebounceTimer: null
    }
  },
  computed: {
    infiniteScrollEnabled() {
      return this.$parent ? this.$parent.infiniteScrollEnabled : true
    },
    cardSize: {
      get () {
        return this.$store.state.sceneList.filters.cardSize
      },
      set (value) {
        this.$store.state.sceneList.filters.cardSize = value
      }
    },
    cardSizeClass () {
      switch (this.$store.state.sceneList.filters.cardSize) {
        case '1':
          return 'is-one-sixth'
        case '2':
          return 'is-one-fifth'
        case '3':
          return 'is-one-quarter'
        case '4':
          return 'is-one-third'
        default:
          return 'is-one-fifth'
      }
    },
    dlState: {
      get () {
        return this.$store.state.sceneList.filters.dlState
      },
      set (value) {
        this.$store.state.sceneList.filters.dlState = value

        switch (this.$store.state.sceneList.filters.dlState) {
          case 'any':
            this.$store.state.sceneList.filters.isAvailable = null
            this.$store.state.sceneList.filters.isAccessible = null
            this.$store.state.sceneList.filters.isHidden = false
            break
          case 'available':
            this.$store.state.sceneList.filters.isAvailable = true
            this.$store.state.sceneList.filters.isAccessible = true
            this.$store.state.sceneList.filters.isHidden = false
            break
          case 'downloaded':
            this.$store.state.sceneList.filters.isAvailable = true
            this.$store.state.sceneList.filters.isAccessible = null
            this.$store.state.sceneList.filters.isHidden = false
            break
          case 'missing':
            this.$store.state.sceneList.filters.isAvailable = false
            this.$store.state.sceneList.filters.isAccessible = null
            this.$store.state.sceneList.filters.isHidden = false
            break
          case 'hidden':
            this.$store.state.sceneList.filters.isAvailable = null
            this.$store.state.sceneList.filters.isAccessible = null
            this.$store.state.sceneList.filters.isHidden = true
            break
        }

        this.reloadList()
      }
    },
    isLoading () {
      return this.$store.state.sceneList.isLoading
    },
    items () {
      return this.$store.state.sceneList.items
    },
    total () {
      return this.$store.state.sceneList.total
    },
    counts () {
      return this.$store.state.sceneList.counts
    },
    show_scene_id() {
      if (this.$store.state.sceneList.show_scene_id != undefined && this.$store.state.sceneList.show_scene_id !='')
      {
        ky.get('/api/scene/'+this.$store.state.sceneList.show_scene_id).json().then(data => {
          if (data.id != 0){
            this.$store.commit('overlay/showDetails', { scene: data })
          }          
        })
        this.$store.state.sceneList.show_scene_id = ''
      }
      
      return this.$store.state.sceneList.show_scene_id
    }
  },
  watch: {
    infiniteScrollEnabled(newVal) {
      if (newVal && this.$refs.infiniteScrollTrigger && !this.infiniteScrollObserver) {
        this.setupInfiniteScroll()
      } else if (!newVal && this.infiniteScrollObserver) {
        this.cleanupInfiniteScroll()
      }
    }
  },
  mounted() {
    this.setupInfiniteScroll()
  },
  beforeDestroy() {
    this.cleanupInfiniteScroll()
    if (this.loadMoreDebounceTimer) {
      clearTimeout(this.loadMoreDebounceTimer)
    }
  },
  methods: {
    setupInfiniteScroll() {
      if (!this.$refs.infiniteScrollTrigger) return
      
      this.infiniteScrollObserver = new IntersectionObserver(
        (entries) => {
          entries.forEach(entry => {
            if (entry.isIntersecting && this.infiniteScrollEnabled && !this.isLoading && this.items.length < this.total) {
              this.loadMore()
            }
          })
        },
        {
          rootMargin: '500px' // Start loading when trigger is 500px from viewport (increased from 300px)
        }
      )
      
      this.infiniteScrollObserver.observe(this.$refs.infiniteScrollTrigger)
    },
    cleanupInfiniteScroll() {
      if (this.infiniteScrollObserver) {
        this.infiniteScrollObserver.disconnect()
        this.infiniteScrollObserver = null
      }
    },
    reloadList () {
      this.$router.push({
        name: 'scenes',
        query: {
          q: this.$store.getters['sceneList/filterQueryParams']
        }
      })
    },
    async loadMore () {
      // Clear any existing debounce timer
      if (this.loadMoreDebounceTimer) {
        clearTimeout(this.loadMoreDebounceTimer)
      }
      
      // Store current scroll position
      const scrollPosition = window.scrollY
      
      // Set a new debounce timer (300ms delay)
      this.loadMoreDebounceTimer = setTimeout(async () => {
        await this.$store.dispatch('sceneList/load', { offset: this.$store.state.sceneList.offset })
        
        // Restore scroll position after content is loaded
        window.scrollTo(0, scrollPosition)
      }, 300)
    }
  }
}
</script>

<style scoped>
  .list-header-label {
    padding-right: 1em;
  }

  /* Add Bulma-style one-sixth column */
  .column.is-one-sixth {
    flex: none;
    width: 16.66666%;
  }

  @media screen and (max-width: 768px) {
    .column.is-one-sixth {
      width: 50%;
    }
  }

  .infiniteScroll-trigger {
    height: 1px;
    width: 100%;
  }
</style>
