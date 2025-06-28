<template>
  <div class="container is-fluid">
    <div class="columns">

      <div class="column is-one-fifth">
        <Filters/>

        <div id="scrollButtons">
          <a id="toTop">
            <b-icon pack="mdi" icon="navigation" />
          </a>
          <a id="toggleInfiniteScroll" @click="toggleInfiniteScroll" :title="infiniteScrollEnabled ? 'Disable Auto Load More' : 'Enable Auto Load More'">
            <b-icon pack="mdi" :icon="infiniteScrollEnabled ? 'reload' : 'pause'" />
          </a>
        </div>
      </div>

      <div class="column is-four-fifths">
        <List :infinite-scroll-enabled="infiniteScrollEnabled"/>
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
      infiniteScrollEnabled: true
    }
  },
  methods: {
    toggleInfiniteScroll() {
      this.infiniteScrollEnabled = !this.infiniteScrollEnabled
    }
  },
  mounted () {
    const toTop = document.getElementById('toTop')
    const toggleBtn = document.getElementById('toggleInfiniteScroll')
    addEventListener('scroll', function () {
      const show = document.body.scrollTop > 20 || document.documentElement.scrollTop > 20
      toTop.style.display = show ? 'block' : 'none'
      toggleBtn.style.display = show ? 'block' : 'none'
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
}
</script>

<style scoped>
  #scrollButtons {
    display: flex;
    justify-content: space-between;
    position: fixed;
    bottom: 20px;
    left: 30px;
    width: 18.5%;
  }
  #toTop, #toggleInfiniteScroll {
    display: none;
    background-color: #f0f0f0;
    color: #4a4a4a;
    padding: 15px;
    border-radius: 10px;
    font-size: 18px;
    margin-right: 8px;
  }
  #toTop:hover, #toggleInfiniteScroll:hover {
    background-color: #BDBDBD;
  }
</style>
