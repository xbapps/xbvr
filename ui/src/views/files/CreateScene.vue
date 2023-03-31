<template>
  <div class="modal is-active">
    <div class="modal-background"></div>
    <div class="modal-card">
      <header class="modal-card-head">
        <p class="modal-card-title">{{ $t("Create Custom Scene") }}</p>
        <button class="delete" @click="close" aria-label="close"></button>
      </header>
      <section class="modal-card-body">
        <div>
          <h6 class="title is-6">{{ file.filename }}</h6>
          <small>
            <span class="pathDetails">{{ file.path }}</span>
            <br/>
            {{ prettyBytes(file.size) }}, {{ file.video_width }}x{{ file.video_height }},
            {{ format(parseISO(file.created_time), "yyyy-MM-dd") }}
          </small>
          <b-field :label="$t('Scene Id')" label-position="on-border" grouped>
            <b-tooltip label="If blank a Scene Id will be generated but cannot be changed later"  :delay="500" >
              <b-input v-model="sceneId" placeholder="Can be empty" ref="sceneIdInput"></b-input>
            </b-tooltip>
          </b-field>
          <b-field :label="$t('Title')" label-position="on-border">            
            <b-input v-model='title' ></b-input>            
          </b-field>
          <b-button class="button is-primary" style="margin-right:1em" v-on:click="addScene(false)">{{$t('Create')}}</b-button>            
          <b-button class="button is-primary" v-on:click="addScene(true)">{{$t('Create/Edit')}} </b-button>            
        </div>
      </section>
    </div>
  </div>
</template>

<script>
import ky from 'ky'
import { format, parseISO } from 'date-fns'
import prettyBytes from 'pretty-bytes'

export default {
  name: 'CreateScene',  
  data () {
    return {
      title: '',
      sceneId: '',
      format,
      parseISO
    }
  },
  computed: {
    file () {
      return this.$store.state.overlay.createScene.file
    }
  },
  mounted () {
    this.initView()
  },
  methods: {
    initView () {
      const commonWords = [
        '180', '180x180', '2880x1440', '3d', '3dh', '3dv', '30fps', '30m', '360',
        '3840x1920', '4k', '5k', '5400x2700', '60fps', '6k', '7k', '7680x3840',
        '8k', 'fb360', 'fisheye190', 'funscript', 'h264', 'h265', 'hevc', 'hq', 'hsp', 'lq', 'lr',
        'mkv', 'mkx200', 'mkx220', 'mono', 'mp4', 'oculus', 'oculus5k',
        'oculusrift', 'original', 'rf52', 'smartphone', 'srt', 'ssa', 'tb', 'uhq', 'vrca220', 'vp9'
      ]
      const isNotCommonWord = word => !commonWords.includes(word.toLowerCase()) && !/^[0-9]+p$/.test(word)

      this.title = (
        this.file.filename
          .replace(/\.|_|\+|-/g, ' ').replace(/\s+/g, ' ').trim()
          .split(' ').filter(isNotCommonWord).join(' ')
          .replace(/ s /g, '\'s '))
      this.$refs.sceneIdInput.focus()
    },
    close () {
      this.$store.commit('overlay/hideCreateCustomScene')
    },
    toInt (value, radix, defaultValue) {
      return parseInt(value, radix || 10) || defaultValue || 0
    },
    addScene(showEdit) {      
      ky.post('/api/scene/create', { json: { title: this.title, id: this.sceneId, filename: this.file.filename } })
        .json()
        .then(scene => {          
          ky.post('/api/files/match', { json: {file_id: this.file.id, scene_id: scene.scene_id}})          
          .then(data => {
            this.$store.dispatch('files/load')
            this.close()
            if (showEdit) {
              this.$store.commit('overlay/editDetails', { scene: scene })
            }
          })          
        })
    },
    prettyBytes
  }
}
</script>

<style scoped>
h6.title.is-6 {
  margin-bottom: 0;
}

h6 + small {
  margin-bottom: 1.5rem;
  display: inline-block;
  font-size: small;
}

h6 + small > .pathDetails {
  color: #B0B0B0;
}

.modal-card {
  position: absolute;
  top: 4em;
  width: 80%;
}

</style>
