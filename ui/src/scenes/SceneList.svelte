<script>
  import ky from "ky";
  import { onMount, afterUpdate } from "svelte";
  import { parse, format } from "date-fns";
  import { cardSize, showInfo, dlState, tag, cast, site } from "../store/filters.js"
  import Video from "./Video.svelte";

  let items = [];
  let data = {};

  let is_available = "1";
  let is_accessible = "1";

  let modal;

  dlState.subscribe(value => {
    items = [];
    getData();
  });
  tag.subscribe(value => {
    items = [];
    getData();
  });
  cast.subscribe(value => {
    items = [];
    getData();
  });
  site.subscribe(value => {
    items = [];
    getData();
  });

  async function getData() {
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
          offset: 0,
          limit: 128,
          is_available: is_available,
          is_accessible: is_accessible,
          tag: $tag,
          cast: $cast,
          site: $site,
        }
      })
      .json();
    items = data.scenes;
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
  onMount(getData);
</script>

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
