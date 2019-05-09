<script>
  import SceneList from "./SceneList.svelte";
  import Filters from "./Filters.svelte";
</script>

<div class="container is-fluid">
  <div class="columns">

    <div class="column is-one-fifth">
      <Filters/>
    </div>

    <div class="column">
        <div class="columns is-multiline">
          <SceneList />
        </div>
    </div>

  </div>
</div>
