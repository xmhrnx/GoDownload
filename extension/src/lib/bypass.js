/**
 * Short-lived state that lets specific downloads through to the browser:
 *  - the bypass set: URLs we re-started in the browser after a failed hand-off
 *    (loop protection: without it we would intercept our own retry);
 *  - the one-shot "Send next download to browser" toggle.
 * Both live in session storage so a service-worker restart doesn't lose them.
 */
import { sessionGet, sessionSet } from './session-store.js';

const BYPASS_KEY = 'bypass';
const SKIP_NEXT_KEY = 'skipNext';
const BYPASS_TTL_MS = 60_000;

async function loadBypass() {
  const now = Date.now();
  const stored = await sessionGet(BYPASS_KEY, {});
  return Object.fromEntries(Object.entries(stored ?? {}).filter(([, exp]) => exp > now));
}

/** Lets the next download of any of these URLs through, for a minute. */
export async function addBypass(...urls) {
  const map = await loadBypass();
  const exp = Date.now() + BYPASS_TTL_MS;
  for (const url of urls) if (url) map[url] = exp;
  await sessionSet(BYPASS_KEY, map);
}

/** True (and forgets the entries) if any of the URLs is bypassed. */
export async function consumeBypass(...urls) {
  const map = await loadBypass();
  const hits = urls.filter((u) => u && u in map);
  if (!hits.length) return false;
  for (const url of hits) delete map[url];
  await sessionSet(BYPASS_KEY, map);
  return true;
}

export async function getSkipNext() {
  return (await sessionGet(SKIP_NEXT_KEY, false)) === true;
}

export async function setSkipNext(value) {
  await sessionSet(SKIP_NEXT_KEY, value === true);
}

/** Returns true once if the toggle was armed, disarming it. */
export async function consumeSkipNext() {
  if (!(await getSkipNext())) return false;
  await setSkipNext(false);
  return true;
}
