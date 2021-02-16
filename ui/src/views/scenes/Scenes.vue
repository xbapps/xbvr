<template>
  <div class="container is-fluid">
    <div class="columns">

      <div class="column is-one-fifth">
        <Filters/>

        <a id="toTop">
          <b-icon pack="mdi" icon="navigation" />
        </a>
      </div>

      <List/>
      <Details v-if="showOverlay"/>
      <EditScene v-if="showEdit" />

    </div>
  </div>
</template>

<script>
import Filters from './Filters'
import List from './List'
import Details from './Details'
import EditScene from './EditScene'

export default {
  name: 'Scenes',
  components: { Filters, List, Details, EditScene },
  mounted () {
    const toTop = document.getElementById('toTop')
    addEventListener('scroll', function () {
      toTop.style.display = document.body.scrollTop > 20 || document.documentElement.scrollTop > 20
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
      vm.$store.dispatch('sceneList/load', { offset: 0 })
    })
  },
  beforeRouteUpdate (to, from, next) {
    if (to.query !== undefined) {
      this.$store.commit('sceneList/stateFromQuery', to.query)
    }
    this.$store.dispatch('sceneList/load', { offset: 0 })
    next()
  },
  computed: {
    showOverlay () {
      return this.$store.state.overlay.details.show
    },
    showEdit () {
      return this.$store.state.overlay.edit.show
    }
  }
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
</style>
