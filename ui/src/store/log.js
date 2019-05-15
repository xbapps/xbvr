import { writable } from 'svelte/store';

export let lockRescan = writable(false);
export let lastRescanMessage = writable({});

export let lockScrape = writable(false);
export let lastScrapeMessage = writable({});
