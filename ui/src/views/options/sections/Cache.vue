<template>
  <div class="container">
    <b-loading :is-full-page="false" :active.sync="isLoading"></b-loading>
    <div class="content">
      <h3>{{$t("Cache")}}</h3>
      <hr/>
      <div class="columns">
        <div class="column is-two-thirds">
          <div class="content" v-if="!isLoading">
            <table>
              <tr>
                <td>
                  <p>
                    <strong>Images</strong>
                  </p>
                  <p>
                    Cache of remote images that were requested at least once.
                  </p>
                </td>
                <td nowrap>{{prettyBytes(sizes.images)}}</td>
                <td>
                  <b-button size="is-small" @click="resetCache('images')">Reset</b-button>
                </td>
              </tr>
              <tr>
                <td>
                  <p><strong>Video previews</strong></p>
                  <p>
                    Generated on demand for local files. Remove when you want to generate previews using new settings.
                  </p>
                </td>
                <td nowrap>{{prettyBytes(sizes.previews)}}</td>
                <td>
                  <b-button size="is-small" @click="resetCache('previews')">Reset</b-button>
                </td>
              </tr>
              <tr>
                <td>
                  <p><strong>Search index</strong> <small> - <span v-if="searchInprogress">Indexing In Progress</span> <span v-if="!searchInprogress">{{indexSceneCount}} scenes indexed</span></small></p>
                  <p>
                    Remove search index when facing issues with finding/matching files.
                  </p>
                </td>
                <td nowrap>{{prettyBytes(sizes.searchIndex)}}</td>
                <td>
                  <b-field>
                    <b-button size="is-small" @click="resetCache('searchIndex')">Reset</b-button>
                    <b-button size="is-small" @click="indexRescan" style="margin-left: .25em;">Rescan</b-button>
                  </b-field>
                </td>
              </tr>
              <tr>
                <td>
                  <p><strong>Scene status</strong></p>
                  <p>
                    Refresh scene status when scenes are not marked "available" or "scripted" despite having such files assigned.
                  </p>
                </td>
                <td nowrap></td>
                <td>
                  <b-button type="is-small" @click="taskRefresh">Refresh Scenes</b-button>
                </td>
              </tr>
            </table>
          </div>
        </div>
        <div class="column">
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import ky from 'ky'
import prettyBytes from 'pretty-bytes'

export default {
  name: 'Cache',
  data () {
    return {
      isLoading: true,
      sizes: {},
      indexSceneCount: 0,
      searchInprogress: false,
    }
  },
  async mounted () {
    await this.loadState()
    this.loadSearchState()
  },
  methods: {
    async loadState () {
      this.isLoading = true
      await ky.get('/api/options/state')
        .json()
        .then(data => {
          this.sizes = data.currentState.cacheSize
          this.isLoading = false
        })
    },
    async resetCache (kind) {
      this.isLoading = true
      await ky.delete(`/api/options/cache/reset/${kind}`, { timeout: 30000 })
      await this.loadState()
      await this.loadSearchState()
    },
    taskRefresh: function () {
      ky.get('/api/task/scene-refresh')
    },
    async loadSearchState () {
      this.isLoading = true
      await ky.get('/api/options/state/search')
        .json()
        .then(data => {
          this.indexSceneCount = data.documentCount
          this.searchInprogress = data.inProgress
          this.isLoading = false
        })
    },
    async indexRescan () {
      this.isLoading = true
      await ky.get('/api/task/index')
      this.searchInprogress = true
      this.isLoading = false
    },
    prettyBytes
  }
}
</script>
