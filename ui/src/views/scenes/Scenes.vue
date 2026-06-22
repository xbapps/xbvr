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
      window.scrollTo({ top: 0, behavior: 'smooth' })
    })
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
    justify-content: flex-start;
    gap: 8px;
    position: fixed;
    bottom: 20px;
    left: 30px;
    z-index: 1000;
  }
  #toTop, #toggleInfiniteScroll {
    display: none;
    background-color: #f0f0f0;
    color: #4a4a4a;
    padding: 15px;
    border-radius: 10px;
    font-size: 18px;
    box-shadow: 0 2px 5px rgba(0, 0, 0, 0.3);
    cursor: pointer;
  }
  #toTop:hover, #toggleInfiniteScroll:hover {
    background-color: #BDBDBD;
  }
</style>
