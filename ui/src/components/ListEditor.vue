<template>
  <section>
    <b-field class="row" position="is-centered" v-for="(item, i) in list" :key="`item-${i}`">      
      <b-input v-if = "columnCount > 1" v-for="fieldidx in columnCount" :key="fieldidx" :class="`list-editor-input list-editor-input-${type}-${i}`" :value="item[fieldidx-1]" @blur="blur(i)" 
        :placeholder=getPlaceholder(fieldidx-1) :style="getColumnStyle(fieldidx-1)" />
      <b-input v-if = "columnCount == undefined || columnCount == 1" :class="`list-editor-input list-editor-input-${type}-${i}`" :value="item" @blur="blur(i)" :placeholder=getPlaceholder(1) />      
      <p class="control">
        <!--<b-button type="is-danger" @click="deleteRow(i)">Delete</b-button>-->
        <b-button type="is-light" @click="deleteRow(i)" icon-right="delete" />
      </p>
      <p class="control">
        <a v-if="showUrl" class="button is-light" 
          :title="`Go to $(${item}`" :href="item" target="_blank" rel="noreferrer">
          <b-icon pack="mdi" icon="link" size="is-small" />
        </a>
      </p>
    </b-field>

    <b-field>
      <b-button class="control" type="is-info" icon-right="plus-circle-outline" @click="addRow">{{$t('Add item')}}</b-button>
    </b-field>
  </section>
</template>

<script>
export default {
  name: 'List2Editor',
  props: {
    list: Array,
    type: String,
    blurFn: Function,
    showUrl: Boolean,
    columnCount: Number,
    placeholders: Array,
    columnStyles: Array,
  },
  methods: {
    addRow () {
      this.list.push('')
    },
    deleteRow (i) {
      this.list.splice(i, 1)
    },
    blur (i) {
      this.list[i] = document.querySelector(`.list-editor-input-${this.type}-${i} input`).value
      this.blurFn.call(null)
    },
    getPlaceholder (i) {
      if (this.placeholders == undefined){
        return ""
      }
      if (i+1 > this.placeholders.length ) {
        return ""
      }
      return this.placeholders[i]
    },
    getColumnStyle (i) {
      if (this.columnStyles == undefined){
        return ""
      }
      if (i+1 > this.columnStyles.length ) {
        return ""
      }
      return this.columnStyles[i]      
    },
  }
}
</script>

<style scoped>
  .list-editor-input {
    width: 100%;
  }
</style>
