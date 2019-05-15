<div class="container">
  <div class="tabs is-medium is-boxed">
    <ul>
      <li class="{active=='volumes' ? 'is-active' : ''}"><a on:click="{()=>active='volumes'}">Folders</a></li>
      <li class="{active=='scrape' ? 'is-active' : ''}"><a on:click="{()=>active='scrape'}">Scene data</a></li>
    </ul>
  </div>

  {#if active == "volumes"}
    <Volumes/>
  {/if}

  {#if active == "scrape"}
    <Scrape/>
  {/if}
</div>

<script>
  import Volumes from "./Volumes.svelte";
  import Scrape from "./Scrape.svelte";

  let active = "volumes";
</script>
