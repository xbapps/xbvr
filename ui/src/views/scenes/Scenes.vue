<template>
  <div class="container is-fluid">
    <div class="columns">

      <div class="column is-one-fifth">
        <Filters/>

        <a id="toTop">
          <b-icon pack="mdi" icon="navigation" />
        </a>

        <a id="infiniteScrollToggle" @click="toggleInfiniteScroll" :data-tooltip="infiniteScrollTooltip">
          <b-icon pack="mdi" :icon="infiniteScrollEnabled ? 'refresh' : 'pause'" />
        </a>
      </div>

      <div class="column is-four-fifths">
        <List/>
      </div>

    </div>
  </div>
</template>

<script>
import Filters from './Filters'
import List from './List'

export default {
  name: 'Scenes',
  components: { Filters, List },
  data() {
    return {
      infiniteScrollEnabled: true,
    }
  },
  computed: {
    infiniteScrollTooltip() {
      return this.infiniteScrollEnabled ? 'Disable auto load more' : 'Enable auto load more'
    }
  },
  mounted () {
    const toTop = document.getElementById('toTop')
    const infiniteScrollToggle = document.getElementById('infiniteScrollToggle')
    addEventListener('scroll', function () {
      toTop.style.display = document.body.scrollTop > 20 || document.documentElement.scrollTop > 20
        ? 'block'
        : 'none'
      infiniteScrollToggle.style.display = document.body.scrollTop > 20 || document.documentElement.scrollTop > 20
        ? 'block'
        : 'none'
    })
    toTop.addEventListener('click', function () {
      scrollToTop()
    })

    const scrollToTop = () => {
      const c = document.documentElement.scrollTop || document.body.scrollTop
      if (c > 0) {
        window.requestAnimationFrame(scrollToTop)
        window.scrollTo(0, c - c / 16)
      }
    }
  },
  beforeRouteEnter (to, from, next) {
    next(vm => {
      if (to.query !== undefined) {
        vm.$store.commit('sceneList/stateFromQuery', to.query)
      }
      vm.$store.dispatch('optionsWeb/load')
      vm.$store.dispatch('sceneList/load', { offset: 0 })
      vm.$store.dispatch('optionsAdvanced/load')
    })
  },
  beforeRouteUpdate (to, from, next) {
    if (to.query !== undefined) {
      this.$store.commit('sceneList/stateFromQuery', to.query)
    }
    this.$store.dispatch('sceneList/load', { offset: 0 })
    next()
  },
  methods: {
    toggleInfiniteScroll() {
      this.infiniteScrollEnabled = !this.infiniteScrollEnabled
      if (this.$refs.listComponent) {
        this.$refs.listComponent.infiniteScrollEnabled = this.infiniteScrollEnabled
      }
    }
  },
}
</script>

<style scoped>
  #toTop {
    display: none;
    position: fixed;
    bottom: 20px;
    left: 30px;
    background-color: #f0f0f0;
    color: #4a4a4a;
    padding: 15px;
    border-radius: 10px;
    font-size: 18px;
  }

  #toTop:hover {
    background-color: #BDBDBD;
  }

  #infiniteScrollToggle {
    display: none;
    position: fixed;
    bottom: 20px;
    left: calc(20% - 60px);
    background-color: #f0f0f0;
    color: #4a4a4a;
    padding: 15px;
    border-radius: 10px;
    font-size: 18px;
    cursor: pointer;
  }

  #infiniteScrollToggle:hover {
    background-color: #BDBDBD;
  }

  #infiniteScrollToggle:hover::after {
    content: attr(data-tooltip);
    position: absolute;
    bottom: 100%;
    right: 0;
    background-color: #333;
    color: white;
    padding: 5px 10px;
    border-radius: 5px;
    font-size: 12px;
    white-space: nowrap;
    margin-bottom: 5px;
    z-index: 1000;
  }
</style>
