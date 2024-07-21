<template>
  <a class="button is-dark is-outlined is-small"
    @click="rescrapeScene()"
    :title="'Rescrape Scene'">
    <b-icon pack="mdi" icon="web-refresh" size="is-small"/>
  </a>
</template>

<script>
import ky from 'ky'
export default {
  name: 'RescrapeButton',
  props: { item: Object },
  methods: {
    delay(ms) {
      return new Promise(resolve => setTimeout(resolve, ms));
    },
    async rescrapeScene () {
      let site = ""

      this.$store.commit('sceneList/toggleSceneList', {scene_id: this.item.scene_id, list: 'needs_update'})
      if (this.item.scraper_id && this.item.needs_update) {
        await this.delay(200);
        ky.get(`/api/task/scrape?site=${this.item.scraper_id}`)
      } else {
        if (this.item.scene_url.toLowerCase().includes("dmm.co.jp")) {
          ky.post('/api/task/scrape-javr', { json: { s: "r18d", q: this.item.scene_id } })
        } else {

          const sites = await ky.get('/api/options/sites').json()
          console.info(sites)

          for (const element of sites) {
            if (this.item.scene_url.toLowerCase().includes(element.id)) {
              site = element.id
            }
          }

          if (this.item.scene_url.toLowerCase().includes("sexlikereal.com")) {
            site = "slr-single_scene"
          }
          if (this.item.scene_url.toLowerCase().includes("czechvrnetwork.com")) {
            site = "czechvr-single_scene"
          }
          if (this.item.scene_url.toLowerCase().includes("povr.com")) {
            site = "povr-single_scene"
          }
          if (this.item.scene_url.toLowerCase().includes("vrporn.com")) {
            site = "vrporn-single_scene"
          }
          if (this.item.scene_url.toLowerCase().includes("vrphub.com")) {
            site = "vrphub-single_scene"
          }
          if (site == "") {
            this.$buefy.toast.open({message: `No scrapers exist for this domain`, type: 'is-danger', duration: 5000})      
            return
          }    
          ky.post(`/api/task/singlescrape`, {timeout: false, json: { site: site, sceneurl: this.item.scene_url, additionalinfo:[] }})
        }
      }
    }
  }
}
</script>
