<template>
  <div>
    <Navbar/>
    <div class="navbar-pad">
      <router-view/>
    </div>
    <Socket/>
  </div>
</template>

<script>
  import Navbar from "@/Navbar.vue";
  import Socket from "@/Socket.vue";

  export default {
    components: {Navbar, Socket},
  }
</script>

<style>
  .b-radio.radio {
    margin-right: 0 !important;
  }
  .navbar-pad {
    margin-top: 1em;
  }
</style>
