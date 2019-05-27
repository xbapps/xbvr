<script>
  import ky from "ky";
  import { onMount, afterUpdate } from "svelte";
  import { parse, format } from "date-fns";
  import { cardSize, showInfo, dlState, tag, cast, site, release_month } from "../store/filters.js"
  import Video from "./Video.svelte";

  let items = [];
  let data = {};

  let offset = 0;
  let limit = 80;
  let total = 0;

  let is_available = "1";
  let is_accessible = "1";

  let modal;

  async function getData(iOffset) {
    switch ($dlState) {
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

    data = await ky
      .get(`/api/scene/list`, {
        searchParams: {
          offset: iOffset,
          limit: limit,
          is_available: is_available,
          is_accessible: is_accessible,
          tag: $tag,
          cast: $cast,
          site: $site,
          released: $release_month,
        }
      })
      .json();

    if (iOffset === 0) {
      items = [];
    }

    items = items.concat(data.scenes);
    offset = iOffset + limit;
    total = data.results;
  }

  function getSizeClass(cardSize) {
    switch (cardSize) {
      case 1:
        return "is-one-fifth";
      case 2:
        return "is-one-quarter";
      case 3:
        return "is-one-third";
    }
  }

  function getImageURL(u) {
    if (u.startsWith("http")) {
      return "/img/700x/" + u.replace("://", ":/");
    } else {
      return u;
    }
  }

  function play(scene) {
    modal = new Video({
    	target: document.body,
    	props: {
    		fileId: scene.file[0].id,
    	}
    });
    modal.$on("close-modal", e => {
      modal.$destroy();
    });
  }

  onMount(() => {
    getData(offset);

      dlState.subscribe(value => {
        offset = 0;
        getData(offset);
      });
      tag.subscribe(value => {
        offset = 0;
        getData(offset);
      });
      cast.subscribe(value => {
        offset = 0;
        getData(offset);
      });
      site.subscribe(value => {
        offset = 0;
        getData(offset);
      });
      release_month.subscribe(value => {
        offset = 0;
        getData(offset);
      });
  });
</script>

<div class="column">
  <div class="columns is-multiline is-full">
    <div class="column">
      <strong>{total} results</strong>
    </div>
  </div>

  <div class="columns is-multiline">

    {#each items as item}
    <div class="column is-multiline {getSizeClass($cardSize)}">
      <div class="card">

        <div class="card-image">
          {#if item.is_available}
          <figure class="image" on:click="{() => item.is_accessible && play(item)}">
            <img src={getImageURL(item.cover_url)} alt="" />
          </figure>
          {:else}
          <figure class="image">
            <img src={getImageURL(item.cover_url)} alt="" style="opacity:0.35;" />
          </figure>
          {/if}
        </div>

        {#if $showInfo}
        <time datetime="">{format(parse(item.release_date), "YYYY-MM-DD")}</time>
        {/if}

      </div>
    </div>
    {/each}

  </div>

  {#if items.length < total}
  <div class="column is-full">
      <a class="button is-fullwidth" on:click={()=>getData(offset)}>Load more</a>
  </div>
  {/if}

</div>

