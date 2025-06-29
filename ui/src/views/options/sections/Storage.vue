<template>
  <div class="content">
    <div class="columns">
      <div class="column">
        <h3 class="title">{{$t('Storage')}}</h3>
      </div>
      <div class="column buttons" align="right">
        <a class="button is-primary" v-on:click="taskRescan">{{ $t('Rescan all folders') }}</a>
      </div>
    </div>
    <div v-if="items.length > 0">
      <b-table :data="items"
               ref="table" default-sort="is_available" default-sort-direction="desc">
        <b-table-column field="path" :label="$t('Path')" sortable v-slot="props">
          {{ props.row.path }}
        </b-table-column>
        <b-table-column field="type" :label="$t('Type')" sortable v-slot="props">
          <b-icon pack="mdi" icon="cloud-outline" size="is-small" v-if="props.row.type !== 'local'"/>
          <b-icon pack="mdi" icon="folder-outline" size="is-small" v-else/>
        </b-table-column>
        <b-table-column field="is_available" :label="$t('Avail')" sortable v-slot="props">
          <b-icon pack="fas" icon="check" size="is-small" v-if="props.row.is_available"></b-icon>
        </b-table-column>
        <b-table-column field="file_count" :label="$t('# of files')" sortable v-slot="props">
          {{ props.row.file_count }}
        </b-table-column>
        <b-table-column field="unmatched_count" :label="$t('Not matched')" sortable v-slot="props">
          {{ props.row.unmatched_count }}
        </b-table-column>
        <b-table-column field="total_size" :label="$t('Total size')" sortable v-slot="props">
          {{ prettyBytes(props.row.total_size) }}
        </b-table-column>
        <b-table-column field="last_scan" :label="$t('Last scan')" sortable v-slot="props">
            <span v-if="props.row.last_scan !== '0001-01-01T00:00:00Z'">
              {{ formatDistanceToNow(parseISO(props.row.last_scan)) }} ago
            </span>
          <span v-else>never</span>
        </b-table-column>
        <b-table-column field="actions" v-slot="props">
          <b-field grouped>
            <button class="button is-small is-outlined" v-on:click='rescanFolder(props.row)' style="margin-right:1em" :title="$t('rescan folder')">
              <b-icon pack="mdi" icon="folder-refresh-outline"></b-icon>
            </button>
            <button class="button is-danger is-small is-outlined" v-on:click='removeFolder(props.row)' :title="$t('remove folder')">
              <b-icon pack="mdi" icon="close-circle" size="is-small"></b-icon>
            </button>
          </b-field>
        </b-table-column>
        <template slot="footer">
          <td></td>
          <td></td>
          <td></td>
          <td>{{ total.files }}</td>
          <td>{{ total.unmatched }}</td>
          <td>{{ prettyBytes(total.size) }}</td>
          <td></td>
          <td></td>
        </template>
      </b-table>
    </div>
    <div v-else>
      <section class="hero">
        <div class="hero-body">
          <div class="container has-text-centered">
            <h1 class="title">
              <span class="icon">
                <b-icon pack="mdi" icon="folder-outline" size="is-large"></b-icon>
              </span>
            </h1>
            <h2 class="subtitle">
              {{ $t('Add folders with VR videos') }}
            </h2>
          </div>
        </div>
      </section>
    </div>

    <hr/>

    <div class="columns">
      <div class="column">
        <h3 class="title">{{ $t('Add local folder') }}</h3>
        <div class="field">
          <label class="label">{{ $t('Path to folder with content') }}</label>
          <div class="control">
            <input class="input" type="text" v-model='newVolumePath'>
          </div>
        </div>
        <div class="control">
          <button class="button is-link" v-on:click='addFolder'>{{ $t('Add new folder') }}</button>
        </div>
      </div>
      <div class="column">
        <h3 class="title">{{ $t('Add cloud storage') }}</h3>
        <b-field grouped>
          <b-field :label="$t('Service')">
            <b-select placeholder="Select one" v-model="serviceSelected">
              <option v-for="option in serviceOpts" :value="option.id" :key="option.id">
                {{ option.name }}
              </option>
            </b-select>
          </b-field>
          <b-field :label="$t('Token')" expanded>
            <b-input v-model='serviceToken'/>
          </b-field>
        </b-field>
        <div class="control">
          <button class="button is-link" v-on:click='addCloudStorage'
                  :disabled="serviceSelected === null || serviceToken === ''">{{ $t('Add service') }}
          </button>
        </div>
      </div>
    </div>

    <hr/>

  <div>
    <h3 class="title">{{ $t('Options') }}</h3>
    <b-field>
      <b-switch v-model="match_ohash" type="is-default">
        Match StashDB Hashes
      </b-switch>
    </b-field>

    <hr/>

    <b-field label="Video File Extensions">
      <b-tooltip label="Only add video file extensions!" position="is-top" style="width: 100%;">
        <b-taginput
            ref="videoExtInput"
            v-model="video_ext"
            :allow-new="true"
            @add="OnExtAdded"
            @remove="saveExtensions"
            placeholder="Add a video extension (e.g. .mp4)">
        </b-taginput>
      </b-tooltip>
    </b-field>
    <b-field>
      <b-tooltip label="This only resets the Video File Extensions list." position="is-right">
        <b-button type="is-warning" @click="resetToDefaults">Reset</b-button>
      </b-tooltip>
    </b-field>
  </div>
  </div>
</template>

<script>
import ky from 'ky'
import prettyBytes from 'pretty-bytes'
import { formatDistanceToNow, parseISO } from 'date-fns'

export default {
  name: 'Storage',
  data () {
    return {
      volumes: [],
      serviceOpts: [{ name: 'Put.io', id: 'putio' }],
      serviceToken: '',
      serviceSelected: null,
      newVolumePath: '',
      prettyBytes,
      parseISO,
      formatDistanceToNow,
      lastAddedTag: null,
      lastAddedTime: 0
    }
  },
  mounted () {
    this.$store.dispatch('optionsStorage/load')
  },
  methods: {
    async taskRescan () {
      try {
        await this.$store.dispatch('optionsStorage/save');
        await ky.get('/api/task/rescan');
      } catch (e) {
        this.$buefy.dialog.alert({
          title: 'Error',
          message: 'Failed to save options. Rescan was not started.',
          type: 'is-danger',
          hasIcon: true,
          ariaRole: 'alertdialog',
          ariaModal: true
        });
      }
    },
    addFolder: async function () {
      await ky.post('/api/options/storage', { json: { path: this.newVolumePath, type: 'local' } })
    },
    addCloudStorage: async function () {
      await ky.post('/api/options/storage', { json: { token: this.serviceToken, type: this.serviceSelected } })
    },
    removeFolder: function (folder) {
      this.$buefy.dialog.confirm({
        title: this.$t('Remove folder'),
        message: `You're about to remove storage location <strong>${folder.path}</strong> and its files from local database - files will remain intact at the location.`,
        type: 'is-danger',
        hasIcon: true,
        onConfirm: function () {
          ky.delete(`/api/options/storage/${folder.id}`)
        }
      })
    },
    rescanFolder: function (folder) {
      ky.get(`/api/task/rescan/${folder.id}`)
    },
    saveExtensions () {
      this.$store.dispatch('optionsStorage/save')
    },
    OnExtAdded(tag) {
      // Debounce the add event as it also triggers on blur
      const now = Date.now();
      if (this.lastAddedTag === tag && now - this.lastAddedTime < 200) {
        return;
      }
      this.lastAddedTag = tag;
      this.lastAddedTime = now;

      // Always remove the raw tag to sanitize in logic below
      const index = this.video_ext.indexOf(tag);
      if (index > -1) {
        this.video_ext.splice(index, 1);
      }

      // Trim whitespace and convert to lowercase
      let cleanTag = tag.trim().toLowerCase();

      // Reject if it contains whitespace in the middle
      if (/\s/.test(cleanTag)) {
        this.$buefy.dialog.alert({
          title: 'Invalid Extension',
          message: 'File extensions cannot contain whitespace.',
          type: 'is-danger',
          hasIcon: true,
          ariaRole: 'alertdialog',
          ariaModal: true
        });
        // Clear the taginput's input field
        if (this.$refs.videoExtInput) {
          this.$refs.videoExtInput.newTag = '';
        }
        return;
      }

      // Reject if it's empty or just a dot
      if (cleanTag.length === 0 || cleanTag === '.') {
        return;
      }

      // Prepend dot if missing
      if (!cleanTag.startsWith('.')) {
        cleanTag = '.' + cleanTag;
      }

      // Reject if it's a forbidden extension
      if (this.forbidden_video_ext && this.forbidden_video_ext.includes(cleanTag)) {
        this.$buefy.dialog.alert({
          title: 'Invalid Extension',
          message: `The file extension <strong>${cleanTag}</strong> is a reserved extension and cannot be added.`,
          type: 'is-danger',
          hasIcon: true,
          ariaRole: 'alertdialog',
          ariaModal: true
        });
        // Clear the taginput's input field
        if (this.$refs.videoExtInput) {
          this.$refs.videoExtInput.newTag = '';
        }
        return;
      }

      // Only allow extensions that are a dot followed by alphanumeric characters
      if (!/^\.[a-z0-9]+$/.test(cleanTag)) {
        this.$buefy.dialog.alert({
          title: 'Invalid Extension',
          message: 'File extensions must only contain letters and numbers after the dot.',
          type: 'is-danger',
          hasIcon: true,
          ariaRole: 'alertdialog',
          ariaModal: true
        });
        if (this.$refs.videoExtInput) {
          this.$refs.videoExtInput.newTag = '';
        }
        return;
      }

      // Add the sanitized tag back, if not a duplicate
      if (!this.video_ext.includes(cleanTag)) {
        this.video_ext.push(cleanTag);
        this.saveExtensions();
      }
    },
    resetToDefaults () {
      this.video_ext = this.default_video_ext.slice()
      this.saveExtensions()
    },
  },
  computed: {
    match_ohash: {
      get () {        
        return this.$store.state.optionsStorage.options.match_ohash
      },
      set (value) {
        this.$store.state.optionsStorage.options.match_ohash = value
      },
    },
    total () {
      let files = 0; let unmatched = 0; let size = 0
      this.$store.state.optionsStorage.items.map(v => {
        files = files + v.file_count
        unmatched = unmatched + v.unmatched_count
        size = size + v.total_size
      })
      return { files, unmatched, size }
    },
    items () {
      return this.$store.state.optionsStorage.items
    },
    video_ext: {
      get () {return this.$store.state.optionsStorage.options.video_ext},
      set (value) {this.$store.state.optionsStorage.options.video_ext = value},
    },
    forbidden_video_ext: {
      get () {return this.$store.state.optionsStorage.options.forbidden_video_ext}
    },
    default_video_ext: {
      get () {return this.$store.state.optionsStorage.options.default_video_ext}
    },
  }
}
</script>
