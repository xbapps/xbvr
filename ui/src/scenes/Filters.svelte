<script>
  import ky from "ky";
  import {onMount} from "svelte";
  import { cardSize, dlState, tag, cast, site, release_month } from "../store/filters.js"

  let filters = {};

  $dlState = "available";
  let is_available = "1";
  let is_accessible = "1";

  function onStateChange(e) {
    switch (e.target.value) {
      case "any":
        is_available = "";
        is_accessible = "";
        break;
      case "available":
        is_available = "1";
        is_accessible = "1";
        break;
      case "downloaded":
        is_available = "1";
        is_accessible = "";
        break;
      case "missing":
        is_available = "0";
        is_accessible = "";
        break;
    }

    getData();
  }

  async function getData() {
    filters = await ky
      .get(`/api/scene/filters/state`, {
        searchParams: {
          is_available: is_available,
          is_accessible: is_accessible,
        }}).json();

    $cast = "";
    $site = "";
    $tag = "";
    $release_month = "";
  }

  onMount(getData);
</script>


<div class="field">
  <label class="label">Cover size</label>
  <input type=range bind:value={$cardSize} min=1 max=3>
</div>

<div class="field">
  <label class="label">State</label>
  <div class="control is-expanded">
    <div class="select is-fullwidth">
      <select bind:value={$dlState} on:change="{onStateChange}">
        <option value="any">Any</option>
        <option value="available">Available right now</option>
        <option value="downloaded">Downloaded</option>
        <option value="missing">Not downloaded</option>
      </select>
    </div>
  </div>
</div>

<label class="label">Release date</label>
<div class="field has-addons">
  <div class="control is-expanded">
    <div class="select is-fullwidth">
      <select bind:value={$release_month}>
        <option></option>
        {#if filters.release_month}
          {#each filters.release_month.reverse() as t}<option>{t}</option>{/each}
        {/if}
      </select>
    </div>
  </div>
  <div class="control">
    <button type="submit" class="button is-light" on:click="{e=>{$release_month=''}}">
      <span class="icon">
        <i class="fas fa-times" />
      </span>
    </button>
  </div>
</div>

<label class="label">Cast</label>
<div class="field has-addons">
  <div class="control is-expanded">
    <div class="select is-fullwidth">
      <select bind:value={$cast}>
        <option></option>
        {#if filters.cast}
          {#each filters.cast as t}<option>{t}</option>{/each}
        {/if}
      </select>
    </div>
  </div>
  <div class="control">
    <button type="submit" class="button is-light" on:click="{e=>{$cast=''}}">
      <span class="icon">
        <i class="fas fa-times" />
      </span>
    </button>
  </div>
</div>

<label class="label">Site</label>
<div class="field has-addons">
  <div class="control is-expanded">
    <div class="select is-fullwidth">
      <select bind:value={$site}>
        <option></option>
        {#if filters.sites}
          {#each filters.sites as t}<option>{t}</option>{/each}
        {/if}
      </select>
    </div>
  </div>
  <div class="control">
    <button type="submit" class="button is-light" on:click="{e=>{$site=''}}">
      <span class="icon">
        <i class="fas fa-times" />
      </span>
    </button>
  </div>
</div>

<label class="label">Tags</label>
<div class="field has-addons">
  <div class="control is-expanded">
    <div class="select is-fullwidth">
      <select bind:value={$tag}>
        <option></option>
        {#if filters.tags}
          {#each filters.tags as t}<option>{t}</option>{/each}
        {/if}
      </select>
    </div>
  </div>
  <div class="control">
    <button type="submit" class="button is-light" on:click="{e=>{$tag=''}}">
      <span class="icon">
        <i class="fas fa-times" />
      </span>
    </button>
  </div>
</div>
