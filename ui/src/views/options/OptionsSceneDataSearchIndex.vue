<template>
  <div class="content">
    <h3>Scene search index</h3>
    <p>
      Once releases metadata is collected, you should populate search index.<br/>
      This needs to be done whenever new scenes are scraped.
    </p>
    <p>
      Warning: this is CPU-intensive process.
    </p>
    <div class="button is-button is-primary" v-on:click="taskIndex()">Index scenes</div>
  </div>
</template>

<script>
  import ky from "ky";

  export default {
    name: "OptionsSceneDataSearchIndex",
    methods: {
      taskIndex() {
        ky.get(`/api/task/index`);
      },
    }
  }
</script>
