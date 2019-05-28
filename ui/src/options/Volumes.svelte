<div class="columns">
  <div class="column is-two-thirds">

    {#if volumes.length > 0}
    <table class="table">
      <thead>
        <tr>
          <th>Path</th>
          <th>Available</th>
          <th># of files</th>
          <th>Last scan</th>
          <th></th>
        </tr>
      </thead>
      <tbody>
        {#each volumes as v}
        <tr>
          <td>{v.path}</td>
          <td>
            {#if v.is_available}
            <span class="icon">
              <i class="fas fa-check"></i>
            </span>
            {/if}
          </td>
          <td>{v.file_count}</td>
          <td>{distanceInWordsToNow(parse(v.last_scan))} ago</td>
          <td></td>
        </tr>
        {/each}
      </tbody>
    </table>

    <div class="button is-button is-primary" on:click={taskRescan}>Rescan</div>
    {/if}
  </div>

  <div class="column">
    <div class="field">
      <label class="label">Path to folder with content</label>
      <div class="control">
        <input class="input" type="text" bind:value={newVolumePath}>
      </div>
    </div>
    <div class="control">
      <button class="button is-link" on:click={addNewVolume}>Add new folder</button>
    </div>
  </div>
</div>

{#if $lastRescanMessage.message !== undefined}
<div class="columns">
  <div class="column is-full">
    <div class="message">
      <div class="message-body">
        {#if $lockRescan}
        <span class="icon">
          <i class="fas fa-spinner fa-pulse"></i>
        </span>
        {/if}
        {$lastRescanMessage.message}
      </div>
    </div>
  </div>
</div>
{/if}


<script>
  import { lockRescan, lastRescanMessage } from "../store/log.js";
  import { parse, format, distanceInWordsToNow } from "date-fns";
  import { onMount } from "svelte";
  import ky from "ky";

  let volumes = [];

  let newVolumePath = "";

  function taskRescan() {
    ky.get(`/api/task/rescan`);
  }

  async function addNewVolume() {
    resp = await ky.post(`/api/config/volume`, {json: {path: newVolumePath}}).json();
    getData();
  }

  async function getData() {
    volumes = await ky.get(`/api/config/volume`).json();
  }

  lockRescan.subscribe(value => {;
    getData();
  });

  onMount(() => {
    getData();
  });
</script>
