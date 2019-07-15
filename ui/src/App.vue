<template>
  <div>
    <Navbar/>
    <router-view/>
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