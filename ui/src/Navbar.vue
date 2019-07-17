<template>
  <nav class="navbar is-fixed-top" role="navigation" aria-label="main navigation">
    <div class="navbar-brand">
      <a class="navbar-item">
        <h1 class="title">XBVR</h1>
      </a>
    </div>

    <div id="navbarBasicExample" class="navbar-menu">
      <div class="navbar-start">
        <a class="navbar-item">
          <router-link to="./">Scenes</router-link>
        </a>
        <a class="navbar-item">
          <router-link to="./files">Files</router-link>
        </a>
        <a class="navbar-item">
          <router-link to="./options">Options</router-link>
        </a>
      </div>
    </div>
  </nav>
</template>

<script>
  let d = document.documentElement;
  d.className += " has-navbar-fixed-top";
</script>