<nav class="navbar is-fixed-top" role="navigation" aria-label="main navigation">
  <div class="navbar-brand">
  <a class="navbar-item">
    <h1 class="title">XBVR</h1>
    </a>
  </div>

  <div id="navbarBasicExample" class="navbar-menu">
    <div class="navbar-start">
      <a class="navbar-item">
        <Link to="./">Scenes</Link>
      </a>
      <a class="navbar-item">
        <Link to="./options">Options</Link>
      </a>
    </div>
  </div>
</nav>

<script>
  import { Link } from "svelte-routing";

  let d = document.documentElement;
  d.className += " has-navbar-fixed-top";
</script>