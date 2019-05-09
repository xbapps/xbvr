import { writable } from 'svelte/store';

export const cardSize = writable(1);
export const showInfo = writable(false);

export let tag = writable("");
export let cast = writable("");
export let site = writable("");

export let dlState = writable("");
