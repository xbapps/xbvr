<template>
  <div class="modal is-active">
    <div class="modal-background"></div>

    <div class="modal-card">
      <header class="modal-card-head">
        <p class="modal-card-title">{{this.label}}</p>
        <button class="delete" @click="close" aria-label="close"></button>
      </header>

      <section class="modal-card-body">
        <b-field class="row" position="is-centered" v-for="(item, i) in list" :key="`item-${i}`">
          <b-input :class="`list-editor-input list-editor-input-${i}`" :value="item" @blur="blur(i)" />
          <p class="control">
            <b-button type="is-danger" @click="deleteRow(i)">Delete</b-button>
          </p>
        </b-field>

        <b-field grouped>
          <b-button class="control" type="is-info" @click="addRow">{{$t('Add item')}}</b-button>
          <b-button class="control" type="is-primary" @click="save">{{ $t('Save list') }}</b-button>
        </b-field>
      </section>
    </div>
  </div>
</template>

<script>
  export default {
    name: "ListEditor",
    data() {
      return {
        label: this.$store.state.overlay.listEditor.label,
        list: this.$store.state.overlay.listEditor.list,
      };
    },
    methods: {
      close() {
        this.$store.commit("overlay/hideListEditor");
      },
      save() {
        this.$store.commit("overlay/hideListEditor");
      },
      addRow() {
        this.list.push("");
      },
      deleteRow(i) {
        this.list.splice(i, 1);
      },
      blur(i) {
        this.list[i] = document.querySelector(`.list-editor-input-${i} input`).value;
      },
    }
  }
</script>

<style scoped>
  .list-editor-input {
    width: 100%;
  }
</style>
