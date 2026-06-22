<template>
  <div class="container is-fluid">
    <div class="columns">

      <div class="column is-one-fifth">
         <Filters/> 

        <a id="toTop">
          <b-icon pack="mdi" icon="navigation" />
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
  name: 'Actors',  
  components: { Filters, List},
  mounted () {
    const toTop = document.getElementById('toTop')
    addEventListener('scroll', function () {
      toTop.style.display = document.body.scrollTop > 20 || document.documentElement.scrollTop > 20
        ? 'block'
        : 'none'
    })
    toTop.addEventListener('click', function () {
      window.scrollTo({ top: 0, behavior: 'smooth' })
    })
  },
  beforeRouteEnter (to, from, next) {
    next(vm => {
      if (to.query !== undefined) {
        vm.$store.commit('actorList/stateFromQuery', to.query)
      }
      vm.$store.dispatch('optionsWeb/load')
      const page = parseInt(to.query.page) || 1
      const limit = vm.$store.state.actorList.limit
      const offset = (page - 1) * limit
      vm.$store.dispatch('actorList/load', { offset })
      vm.$store.dispatch('optionsAdvanced/load')
    })
  },
  beforeRouteUpdate (to, from, next) {
    if (to.query !== undefined) {
      this.$store.commit('actorList/stateFromQuery', to.query)
    }
    const page = parseInt(to.query.page) || 1
    const limit = this.$store.state.actorList.limit
    const offset = (page - 1) * limit
    this.$store.dispatch('actorList/load', { offset })
    next()
  },
  computed: {
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
