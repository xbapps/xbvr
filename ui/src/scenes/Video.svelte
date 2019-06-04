<div class="modal is-active">
  <div class="modal-background"></div>
  <div class="modal-content">
    <video bind:this={videoElement} width="640" height="640" id="videojs-vr-player" class="video-js vjs-default-skin" controls playsinline autoplay>
      <source src="/api/dms/file/{fileId}" type="video/mp4">
    </video>
  </div>
  <button class="modal-close is-large" aria-label="close" on:click="{() => { player.dispose(); dispatch('close-modal'); }}"></button>
</div>

<script>
  import { onMount, createEventDispatcher } from 'svelte';
  import videojs from "video.js";
  import vr from "videojs-vr";
  import hotkeys from "videojs-hotkeys";

  export let fileId;

  let videoElement;
  let player;

  const dispatch = createEventDispatcher();

  onMount(() => {
    player = videojs(videoElement);
    let vr = player.vr({
      projection: '360',
      forceCardboard: false
    });

    player.hotkeys({
      alwaysCaptureHotkeys: true,
      volumeStep: 0.1,
      seekStep: 5,
      enableModifiersForNumbers: false,
      customKeys: {
        closeModal: {
          key: function(event) {
            return event.which === 27
          },
          handler: function(player, options, event) {
            player.dispose();
            dispatch('close-modal')
          }
        }
      }
    });

    player.on("loadedmetadata", function() {
      vr.camera.position.set(-1, 0, -1);
    });
  });
</script>
