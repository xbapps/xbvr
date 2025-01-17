const state = {
  details: {
    show: false,
    scene: null,
    altsrc: null,
    prevscene: null,
    query_for_altsrc: '',
  },
  edit: {
    show: false,
    scene: null
  },
  player: {
    show: false,
    file: null
  },
  match: {
    show: false,
    file: null
  },
  createScene: {
    show: false,
    file: null
  },
  actordetails: {
    show: false,
    actor: null
  },
  actoredit: {
    show: false,
    actor: null
  },  
  quickFind: {
    show:false,
    searchString: null,  // use to preppoulate the search box
    displaySelectedScene: true,    
    selectedScene: null,        // selected scene 
  },
  sceneMatchParams:{ // overlay to edit matching params
    show:false,
    site: '',
  },
  searchStashDbScenes: {
    show: false,
    scene: null
  },
  searchStashDbActors: {
    show: false,
    actor: null
  },
  changeDetailsTab: -1,
}

const mutations = {
  showDetails (state, payload) {
    state.details.scene = payload.scene
    state.details.altsrc = payload.altsrc
    state.details.prevscene = payload.prevscene
    state.details.query_for_altsrc = payload.query_for_altsrc
    state.details.show = true
  },
  hideDetails (state, payload) {
    if (state.details.altsrc != null) {
      // if we are display scene data from another source, go back to the real scene
      state.details.show = false
      state.details.altsrc = null
      state.details.scene = state.details.prevscene
      state.details.prevscene = null
      state.details.query_for_altsrc = ''
      state.details.show = true
    }else {
      state.details.scene = null
      state.details.show = false
    }
  },
  editDetails (state, payload) {
    state.edit.scene = payload.scene
    state.edit.show = true
  },
  hideEditDetails (state) {
    state.edit.scene = null
    state.edit.show = false
  },
  showActorDetails (state, payload) {
    state.actordetails.actor = payload.actor
    state.actordetails.show = true    
  },
  hideActorDetails (state, payload) {
    state.actordetails.actor = null
    state.actordetails.show = false
  },
  editActorDetails (state, payload) {
    state.actoredit.actor = payload.actor
    state.actoredit.show = true
  },
  hideActorEditDetails (state) {
    state.actoredit.scene = null
    state.actoredit.show = false
  },
  showPlayer (state, payload) {
    state.player.file = payload.file
    state.player.show = true
  },
  hidePlayer (state, payload) {
    state.player.file = null
    state.player.show = false
  },
  showMatch (state, payload) {
    state.match.file = payload.file
    state.match.show = true
  },
  hideMatch (state, payload) {
    state.match.file = null
    state.match.show = false
  },
  createCustomScene (state, payload) {
    state.createScene.file = payload.file
    state.createScene.show = true
  },
  hideCreateCustomScene (state, payload) {
    state.createScene.file = null
    state.createScene.show = false    
  },
  showQuickFind (state, payload) {
    state.quickFind.displaySelectedScene = true    
    state.quickFind.selectedScene = null    
    if (payload !== undefined) {
      if (payload.searchString !== undefined) {
        state.quickFind.searchString = payload.searchString
      }
      if (payload.displaySelectedScene !== undefined) {
        state.quickFind.displaySelectedScene = payload.displaySelectedScene
      }
    }    
    state.quickFind.show = true
  },
  hideQuickFind (state, payload) {
    state.quickFind.show = false
  },
  showSceneMatchParams (state, payload) {
    state.sceneMatchParams.site = payload.site
    state.sceneMatchParams.show = true    
  },
  hideSceneMatchParams (state, payload) {
    state.sceneMatchParams.show = false
  },
  showSearchStashdbScenes (state, payload) {
    state.searchStashDbScenes.scene = payload.item
    state.searchStashDbScenes.show = true
  },
  hideSearchStashdbScenes (state) {
    state.searchStashDbScenes.scene = null
    state.searchStashDbScenes.show = false
  },
  showSearchStashdbActors (state, payload) {
    state.searchStashDbActors.actor = payload.item
    state.searchStashDbActors.show = true
  },
  hideSearchStashdbActors (state) {
    state.searchStashDbActors.actor = null
    state.searchStashDbActors.show = false
  },
  changeDetailsTab (state, payload) {
    state.changeDetailsTab = payload.tab
  },
}

export default {
  namespaced: true,
  state,
  mutations
}
