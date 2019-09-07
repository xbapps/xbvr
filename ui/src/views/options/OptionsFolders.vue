<template>
  <div>
    <div class="columns">
      <div class="column is-two-thirds">
        <div v-if="items.length > 0">
          <b-table :data="items"
                   ref="table" default-sort="is_available" default-sort-direction="desc">
            <template slot-scope="props">
              <b-table-column field="path" label="Path" sortable>
                {{props.row.path}}
              </b-table-column>
              <b-table-column field="is_available" label="Avail" sortable>
                <b-icon pack="fas" icon="check" size="is-small" v-if="props.row.is_available"></b-icon>
              </b-table-column>
              <b-table-column field="file_count" label="# of files" sortable>
                {{props.row.file_count}}
              </b-table-column>
              <b-table-column field="unmatched_count" label="Not matched" sortable>
                {{props.row.unmatched_count}}
              </b-table-column>
              <b-table-column field="total_size" label="Total size" sortable>
                {{prettyBytes(props.row.total_size)}}
              </b-table-column>
              <b-table-column field="last_scan" label="Last scan" sortable>
                {{formatDistanceToNow(parseISO(props.row.last_scan))}} ago
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

          <div class="button is-button is-primary" v-on:click="taskRescan">Rescan</div>
        </div>
      </div>

      <div class="column">
        <div class="field">
          <label class="label">Path to folder with content</label>
          <div class="control">
            <input class="input" type="text" v-model='newVolumePath'>
          </div>
        </div>
        <div class="control">
          <button class="button is-link" v-on:click='addFolder'>Add new folder</button>
        </div>
      </div>
    </div>

    <div class="columns">
      <div class="column is-full">
        <b-message v-if="Object.keys(lastMessage).length !== 0">
          <span class="icon" v-if="lock">
            <i class="fas fa-spinner fa-pulse"></i>
          </span>
          {{lastMessage.message}}
        </b-message>
      </div>
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
          title: 'Remove folder',
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
      lastMessage() {
        return this.$store.state.messages.lastRescanMessage;
      },
      lock() {
        return this.$store.state.messages.lockRescan;
      }
    }
  }
</script>
