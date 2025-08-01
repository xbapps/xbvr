<template>
  <div>
    <div class="field">
      <div class="control">
        <input 
          class="input" 
          type="text" 
          v-model="newItem" 
          :placeholder="$t('Add URL or drag local image files here')" 
          @keyup.enter="addItem"
          @drop="handleFileDrop"
          @dragover.prevent
          @dragenter.prevent
          @dragleave.prevent>
      </div>
    </div>

    <!-- Lock Control -->
    <div style="margin-bottom: 0.25rem; padding: 0;">
      <div style="display: flex; justify-content: space-between; align-items: center; line-height: 1; padding: 0;">
        <span style="font-size: 0.6rem; color: #b5b5b5; line-height: 1; margin: 0;">
          Drag images to reorder
        </span>
        <b-button 
          type="is-light" 
          size="is-small" 
          @click="toggleLock"
          :class="{ 'is-info': !isLocked, 'is-warning': isLocked }"
          icon-left="lock"
          style="font-size: 0.6rem; padding: 0.25rem 0.5rem; line-height: 1; margin: 0;">
          {{ isLocked ? 'Unlock' : 'Lock' }} Delete
        </b-button>
      </div>
    </div>

    <draggable :list="internalList" @end="onDragEnd" class="image-grid">
      <div v-for="(item, index) in internalList" :key="index" class="image-item">
        <img :src="getImageURL(item)" alt="Gallery image" class="gallery-image"/>
        <div class="image-controls">
          <b-tooltip :label="$t('Delete Image')" type="is-dark" position="is-top" :delay="500" append-to-body>
            <b-button 
              type="is-danger" 
              size="is-small" 
              @click="removeItem(index)" 
              icon-left="delete"
              :disabled="isLocked"
              :class="{ 'is-light': isLocked }">
            </b-button>
          </b-tooltip>
          <b-tooltip :label="$t('Set as Cover')" type="is-dark" position="is-top" :delay="500" append-to-body>
            <b-button
              type="is-primary"
              size="is-small"
              :class="{ 'is-light': item !== coverUrl }"
              @click="setCover(item)"
              icon-left="image">
            </b-button>
          </b-tooltip>
        </div>
      </div>
    </draggable>
  </div>
</template>

<script>
import draggable from 'vuedraggable'

export default {
  name: 'GalleryEditor',
  components: {
    draggable
  },
  props: {
    list: {
      type: Array,
      required: true
    },
    coverUrl: {
      type: String,
      default: ''
    },
    blurFn: {
      type: Function,
      default: () => {}
    }
  },
  data () {
    return {
      internalList: [...this.list],
      newItem: '',
      isLocked: true // Default to locked
    }
  },
  watch: {
    list(newList) {
      this.internalList = [...newList];
    }
  },
  methods: {
    toggleLock() {
      this.isLocked = !this.isLocked;
    },
    addItem () {
      if (this.newItem.trim() !== '') {
        this.internalList.push(this.newItem.trim())
        this.newItem = ''
        this.updateList()
      }
    },
    removeItem (index) {
      if (!this.isLocked) {
        const itemToDelete = this.internalList[index]
        
        // Check if this is the cover image
        if (itemToDelete === this.coverUrl) {
          // Show confirmation dialog for deleting cover image
          this.$buefy.dialog.confirm({
            title: 'Delete Cover Image',
            message: 'You are about to delete the current cover image. This will clear the cover selection. Are you sure you want to continue?',
            confirmText: 'Delete',
            cancelText: 'Cancel',
            type: 'is-warning',
            hasIcon: true,
            onConfirm: () => {
              // Remove the image
              this.internalList.splice(index, 1)
              // Clear the cover_url since we're deleting the cover image
              this.$emit('setCover', '')
              this.updateList()
            }
          })
        } else {
          // Regular image deletion - just remove it without affecting cover_url
          this.internalList.splice(index, 1)
          this.updateList()
        }
      }
    },
    updateList () {
      this.$emit('update:list', this.internalList)
      this.blurFn()
    },
    onDragEnd (evt) {
      // 'draggable' updates the list automatically, so we just need to emit the update
      this.updateList()
    },
    setCover (url) {
      this.$emit('setCover', url)
    },
    getImageURL (url) {
      if (!url) return url
      try {
        if (url.startsWith('http')) {
          if (url.indexOf('%') === -1) {
            return '/img/200x/' + encodeURI(url)
          } else {
            return '/img/200x/' + encodeURI(decodeURI(url))
          }
        }
      } catch {
        // fall through
      }
      
      // Convert backslashes to forward slashes
      if (url.includes('\\')) {
        url = url.replace(/\\/g, '/')
      }
      
      // Return path or url
      return url
    },
    handleFileDrop(event) {
      event.preventDefault();
      const files = event.dataTransfer.files;
      for (let i = 0; i < files.length; i++) {
        const file = files[i];
        const reader = new FileReader();
        reader.onload = (e) => {
          const url = e.target.result;
          this.newItem = url;
          this.addItem();
        };
        reader.readAsDataURL(file);
      }
    }
  }
}
</script>

<style scoped>
.image-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(120px, 178px));
  grid-auto-rows: 120px;
  gap: 0.5rem;
  overflow-y: auto;
}

.image-item {
  position: relative;
  overflow: hidden;
  word-break: break-all;
  display: flex;
  align-items: stretch;
  justify-content: center;
  min-height: 120px;
  max-height: 120px;
  min-width: 120px;
  max-width: 178px;
  aspect-ratio: 16/9;
}

.gallery-image {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.image-controls {
  position: absolute;
  bottom: 0.5rem;
  right: 0.5rem;
  display: flex;
  gap: 0.5rem;
  opacity: 0;
  transition: opacity 0.3s ease;
}

.image-item:hover .image-controls {
  opacity: 1;
}

.image-controls .button {
  height: 2rem;
  width: 2rem;
  padding: 0;
}
</style> 