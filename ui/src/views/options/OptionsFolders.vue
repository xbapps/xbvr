<template>
  <div>
    <div class="columns">
      <div class="column is-two-thirds">
        <div v-if="volumes.length > 0">
          <table class="table">
            <thead>
            <tr>
              <th>Path</th>
              <th>Available</th>
              <th># of files</th>
              <th>Not matched</th>
              <th>Total size</th>
              <th>Last scan</th>
              <th></th>
            </tr>
            </thead>
            <tbody>
            <tr v-for="v in volumes" v-bind:key="v.path">
              <td>{{v.path}}</td>
              <td>
                <b-icon pack="fas" icon="check" size="is-small" v-if="v.is_available"></b-icon>
              </td>
              <td>{{v.file_count}}</td>
              <td>{{v.unmatched_count}}</td>
              <td>{{prettyBytes(v.total_size)}}</td>
              <td>{{distanceInWordsToNow(parse(v.last_scan))}} ago</td>
              <td></td>
            </tr>
            </tbody>
          </table>

          <div class="button is-button is-primary" v-on:click="taskRescan()">Rescan</div>
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
          <button class="button is-link" v-on:click='addFolder()'>Add new folder</button>
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
  import {distanceInWordsToNow, parse} from "date-fns";

  export default {
    name: "OptionsFolders",
    data() {
      return {
        volumes: [],
        newVolumePath: "",
        prettyBytes,
        parse,
        distanceInWordsToNow,
      }
    },
    mounted() {
      this.getData();
    },
    methods: {
      getData: async function getData() {
        this.volumes = await ky.get(`/api/config/volume`).json();
      },
      taskRescan: function () {
        ky.get(`/api/task/rescan`);
      },
      addFolder: async function () {
        await ky.post(`/api/config/volume`, {json: {path: this.newVolumePath}}).json();
        this.getData();
      }
    },
    computed: {
      lastMessage() {
        return this.$store.state.messages.lastRescanMessage;
      },
      lock() {
        return this.$store.state.messages.lockRescan;
      }
    }
  }
</script>
