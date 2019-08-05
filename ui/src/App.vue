<template>
  <div>
    <Navbar/>
    <div class="pad">
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

<style scoped>
  .pad {
    margin-top: 1em;
  }
</style>
