<template>
  <div class="content">
    <h3 class="title">{{$t('Folders')}}</h3>
    <div v-if="items.length > 0">
      <b-table :data="items"
               ref="table" default-sort="is_available" default-sort-direction="desc">
        <template slot-scope="props">
          <b-table-column field="path" :label="$t('Path')" sortable>
            {{props.row.path}}
          </b-table-column>
          <b-table-column field="is_available" :label="$t('Avail')" sortable>
            <b-icon pack="fas" icon="check" size="is-small" v-if="props.row.is_available"></b-icon>
          </b-table-column>
          <b-table-column field="file_count" :label="$t('# of files')" sortable>
            {{props.row.file_count}}
          </b-table-column>
          <b-table-column field="unmatched_count" :label="$t('Not matched')" sortable>
            {{props.row.unmatched_count}}
          </b-table-column>
          <b-table-column field="total_size" :label="$t('Total size')" sortable>
            {{prettyBytes(props.row.total_size)}}
          </b-table-column>
          <b-table-column field="last_scan" :label="$t('Last scan')" sortable>
                <span v-if="props.row.last_scan !== '0001-01-01T00:00:00Z'">
                  {{formatDistanceToNow(parseISO(props.row.last_scan))}} ago
                </span>
            <span v-else>never</span>
          </b-table-column>
          <b-table-column field="actions">
            <button class="button is-danger is-small is-outlined" v-on:click='removeFolder(props.row)'>
              <b-icon pack="mdi" icon="close-circle" size="is-small"></b-icon>
            </button>
          </b-table-column>
        </template>
        <template slot="footer">
          <td></td>
          <td></td>
          <td>{{total.files}}</td>
          <td>{{total.unmatched}}</td>
          <td>{{prettyBytes(total.size)}}</td>
          <td></td>
          <td></td>
        </template>
      </b-table>

      <div class="button is-button is-primary" v-on:click="taskRescan">{{$t('Rescan')}}</div>
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
              {{$t('Add folders with VR videos')}}
            </h2>
          </div>
        </div>
      </section>
    </div>

    <hr/>

    <h3 class="title">{{$t('Add folder')}}</h3>
    <div class="field">
      <label class="label">{{$t('Path to folder with content')}}</label>
      <div class="control">
        <input class="input" type="text" v-model='newVolumePath'>
      </div>
    </div>
    <div class="control">
      <button class="button is-link" v-on:click='addFolder'>{{$t('Add new folder')}}</button>
    </div>

  </div>

</template>

<script>
  import ky from "ky";
  import prettyBytes from "pretty-bytes";
  import {formatDistanceToNow, parseISO} from "date-fns";

  export default {
    name: "OptionsFolders",
    data() {
      return {
        volumes: [],
        newVolumePath: "",
        prettyBytes,
        parseISO,
        formatDistanceToNow,
      }
    },
    mounted() {
      this.$store.dispatch("optionsFolders/load");
    },
    methods: {
      taskRescan: function () {
        ky.get(`/api/task/rescan`);
      },
      addFolder: async function () {
        await ky.post(`/api/config/folder`, {json: {path: this.newVolumePath}});
      },
      removeFolder: function (folder) {
        this.$buefy.dialog.confirm({
          title: this.$t('Remove folder'),
          message: `You're about to remove folder <strong>${folder.path}</strong> and its files from database.`,
          type: 'is-danger',
          hasIcon: true,
          onConfirm: function () {
            ky.delete(`/api/config/folder/${folder.id}`);
          }
        });
      }
    },
    computed: {
      total() {
        let files = 0, unmatched = 0, size = 0;
        this.$store.state.optionsFolders.items.map(v => {
          files = files + v.file_count;
          unmatched = unmatched + v.unmatched_count;
          size = size + v.total_size;
        });
        return {files, unmatched, size}
      },
      items() {
        return this.$store.state.optionsFolders.items;
      },
    }
  }
</script>
