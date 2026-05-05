/**
 * Static shim for $app/environment used by Vitest when running Hive unit
 * tests in jsdom. Hive is a client-side PWA; every test assumes a browser
 * context, so browser is unconditionally true.
 */
export const browser = true;
export const building = false;
export const dev = true;
export const version = 'test';
