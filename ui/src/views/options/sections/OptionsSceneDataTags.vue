<template>
  <div class="content">
    <h3 class="title">{{ $t('Promoted tags') }}</h3>
    <p>{{ $t('Tags marked as promoted will be shown as a badge under the scene thumbnail.') }}</p>
    <b-field :label="$t('Search tags')">
      <b-input v-model="search" icon="magnify"/>
    </b-field>
    <b-table :data="filteredTags" :per-page="50" paginated>
      <b-table-column field="name" :label="$t('Tag')" sortable v-slot="props">
        {{ props.row.name }}
      </b-table-column>
      <b-table-column field="count" :label="$t('Scenes')" numeric sortable v-slot="props">
        {{ props.row.count }}
      </b-table-column>
      <b-table-column field="is_promoted" :label="$t('Promoted')" width="120" v-slot="props">
        <b-switch v-model="props.row.is_promoted" @input="togglePromoted(props.row)"></b-switch>
      </b-table-column>
    </b-table>
  </div>
</template>

<script>
import ky from 'ky'

export default {
  name: 'OptionsSceneDataTags',
  data () {
    return {
      tags: [],
      search: ''
    }
  },
  computed: {
    filteredTags () {
      if (!this.search) {
        return this.tags
      }
      const q = this.search.toLowerCase()
      return this.tags.filter(t => t.name.toLowerCase().includes(q))
    }
  },
  mounted () {
    this.loadTags()
  },
  methods: {
    async loadTags () {
      try {
        const resp = await ky.get('/api/tag/list').json()
        this.tags = resp.tags || []
      } catch (error) {
        this.tags = []
        this.$buefy.toast.open({ message: this.$t('Failed to update tag'), type: 'is-danger' })
      }
    },
    togglePromoted (row) {
      ky.post('/api/tag/promote/' + row.id, { json: { is_promoted: row.is_promoted } }).catch(() => {
        row.is_promoted = !row.is_promoted
        this.$buefy.toast.open({ message: this.$t('Failed to update tag'), type: 'is-danger' })
      })
    }
  }
}
</script>
