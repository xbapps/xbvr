<template>
  <div class="modal is-active">
    <div class="modal-background"></div>
    <div class="modal-content">
      <video ref="player"
             width="640" height="640" class="video-js vjs-default-skin"
             controls playsinline autoplay>
        <source :src="sourceUrl" type="video/mp4">
      </video>
    </div>
    <button class="modal-close is-large" aria-label="close"
            @click="close()"></button>
  </div>
</template>

<script>
  import videojs from "video.js";
  import vr from "videojs-vr";
  import hotkeys from "videojs-hotkeys";

  export default {
    name: "Details",
    data() {
      return {
        player: {},
      }
    },
    computed: {
      item() {
        return this.$store.state.detailsOverlay.scene;
      },
      sourceUrl() {
        if (this.$store.state.detailsOverlay.scene.is_available) {
          return "/api/dms/file/" + this.$store.state.detailsOverlay.scene.file[0].id;
        }
        return "";
      }
    },
    mounted() {
      this.player = videojs(this.$refs.player);
      let vr = this.player.vr({
        projection: '360',
        forceCardboard: false
      });

      this.player.hotkeys({
        alwaysCaptureHotkeys: true,
        volumeStep: 0.1,
        seekStep: 5,
        enableModifiersForNumbers: false,
        customKeys: {
          closeModal: {
            key: function (event) {
              return event.which === 27
            },
            handler: (player, options, event) => {
              this.player.dispose();
              this.$store.commit("hideDetailsOverlay");
            }
          }
        }
      });

      this.player.on("loadedmetadata", function () {
        vr.camera.position.set(-1, 0, -1);
      });
    },
    methods: {
      close() {
        this.player.dispose();
        this.$store.commit("hideDetailsOverlay");
      },
    }
  }
</script>

<style scoped>
  .video-js {
    margin: 0 auto;
  }
</style>