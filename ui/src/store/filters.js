import { writable } from 'svelte/store';

export const cardSize = writable(1);

export let tag = writable("");
export let cast = writable("");
export let site = writable("");
export let release_month = writable("");

export let dlState = writable("");
