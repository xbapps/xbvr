<script>
  import { Router, Route, navigate } from "svelte-routing";
  import Navbar from "./Navbar.svelte";
  import Scenes from "./scenes/index.svelte";

  let url = "";
</script>

<Router url="{url}" basepath="/ui/">
  <Route path="scenes" component="{Scenes}" />

  <Navbar/>
</Router>
