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
import videojs from 'video.js'
import vr from 'videojs-vr/dist/videojs-vr.min.js'
import hotkeys from 'videojs-hotkeys'

export default {
  name: 'Details',
  data () {
    return {
      player: {}
    }
  },
  computed: {
    sourceUrl () {
      if (this.$store.state.overlay.player.file) {
        return '/api/dms/file/' + this.$store.state.overlay.player.file.id + '?dnt=true'
      }
      return ''
    }
  },
  mounted () {
    this.player = videojs(this.$refs.player)
    const vr = this.player.vr({
      projection: '180',
      forceCardboard: false
    })

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
            this.player.dispose()
            this.$store.commit('overlay/hidePlayer')
          }
        }
      }
    })

    this.player.on('loadedmetadata', function () {
      vr.camera.position.set(-1, 0, -1)
    })
  },
  methods: {
    close () {
      this.player.dispose()
      this.$store.commit('overlay/hidePlayer')
    }
  }
}
</script>

<style scoped>
  .video-js {
    margin: 0 auto;
  }
</style>
