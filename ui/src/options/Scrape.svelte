<div class="columns">
  <div class="column">
    <p>
      Releases metadata is required for XBVR to function.
      Recommended path is to import bundled data first, then scrape from sites if neccesary.
    </p>
    <hr/>
    <div class="button is-button is-primary" on:click={taskLoadBundle}>Import bundled data</div>
    <div class="button is-button is-primary" on:click={taskScrape}>Scrape</div>
  </div>
</div>

{#if $lastScrapeMessage.message !== undefined}
<div class="columns">
  <div class="column is-full">
    <div class="message">
      <div class="message-body">
        {#if $lockScrape}
        <span class="icon">
          <i class="fas fa-spinner fa-pulse"></i>
        </span>
        {/if}
        {$lastScrapeMessage.message}
      </div>
    </div>
  </div>
</div>
{/if}

<script>
  import { lockScrape, lastScrapeMessage } from "../store/log.js";
  import ky from "ky";

  function taskScrape() {
    ky.get(`/api/task/scrape`);
  }

  function taskLoadBundle() {
    ky.get(`/api/task/import-bundle`);
  }
</script>