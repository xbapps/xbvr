<template>
  <div class="column">
    <b-loading :is-full-page="true" :active.sync="isLoading"></b-loading>

    <div class="columns is-multiline is-full">
      <div class="column">
        <strong>{{total}} results</strong>
      </div>
      <div class="column">
        <b-field>
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
        </b-field>
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

    <div class="columns is-multiline">
      <div :class="['column', 'is-multiline', cardSizeClass]"
           v-for="item in items" :key="item.id">
        <SceneCard :item="item"/>
      </div>
    </div>

    <div class="column is-full" v-if="items.length < total">
      <a class="button is-fullwidth" v-on:click="loadMore()">{{$t('Load more')}}</a>
    </div>

  </div>
</template>

<script>
import SceneCard from './SceneCard'

export default {
  name: 'List',
  components: { SceneCard },
  computed: {
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
          return 'is-one-fifth'
        case '2':
          return 'is-one-quarter'
        case '3':
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
            break
          case 'available':
            this.$store.state.sceneList.filters.isAvailable = true
            this.$store.state.sceneList.filters.isAccessible = true
            break
          case 'downloaded':
            this.$store.state.sceneList.filters.isAvailable = true
            this.$store.state.sceneList.filters.isAccessible = null
            break
          case 'missing':
            this.$store.state.sceneList.filters.isAvailable = false
            this.$store.state.sceneList.filters.isAccessible = null
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
    async loadMore () {
      this.$store.dispatch('sceneList/load', { offset: this.$store.state.sceneList.offset })
    }
  }
}
</script>

<style scoped>
  .list-header-label {
    padding-right: 1em;
  }
</style>
