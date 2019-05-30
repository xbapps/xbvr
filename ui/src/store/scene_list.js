import { writable } from 'svelte/store';
import ky from "ky";

function createSceneList() {
  const { subscribe, set } = writable([]);

  return {
    subscribe,
    append: (old_items, new_items) => {
      set(old_items.concat(new_items))
    },
    toggleList: (allItems, scene_id, list) => {
      let tmp = allItems.map(obj => {
        if (obj.scene_id === scene_id) {
          if (list === "watchlist") {
            obj.watchlist = !obj.watchlist;
          }
          if (list === "favourite") {
            obj.favourite = !obj.favourite;
          }
        }
        return obj
      });

      ky.post(`/api/scene/toggle`, {
        json: {
          scene_id: scene_id,
          list: list,
        }
      });

      set(tmp)
    },
    reset: () => set([])
  };
}

export const items = createSceneList();
