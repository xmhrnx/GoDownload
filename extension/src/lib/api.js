/**
 * HTTP client for the GoDownload desktop app's local API.
 *
 * Must only be used from extension contexts (background, popup, options): the
 * app validates the Origin header, which content scripts cannot provide.
 */
import { extensionVersion } from './browser.js';
import { getSettings, onSettingsChanged } from './settings.js';
import { log } from './log.js';

const PING_TIMEOUT_MS = 800;

/**
 * The ports the app may listen on, in the order it tries them. There is no
 * port setting: the app takes the first one free (Windows often reserves
 * whole port ranges for Hyper-V, WSL or Docker) and the extension finds it.
 * Keep in sync with Ports in internal/server/listen.go.
 */
export const APP_PORTS = Object.freeze([47821, 18821, 28821, 38821, 58821, 8821]);

/** The port the app was last found on. */
let foundPort = null;
const POST_TIMEOUT_MS = 8000;
const PING_CACHE_MS = 5000;

/** Connection states shown in the popup. */
export const Connection = Object.freeze({
  CONNECTED: 'connected',
  OFFLINE: 'offline',          // app not running / unreachable
  UNAUTHORIZED: 'unauthorized', // app running but rejects our token
});

export class ApiError extends Error {
  /**
   * @param {'offline'|'timeout'|'unauthorized'|'http'|'bad-response'} code
   * @param {string} message
   * @param {number} [status]
   */
  constructor(code, message, status = 0) {
    super(message);
    this.name = 'ApiError';
    this.code = code;
    this.status = status;
  }
}

/** @type {{state: string, info: object|null, at: number} | null} */
let pingCache = null;
/** @type {Promise<object> | null} */
let pingInFlight = null;

// A different token means the cached reachability is meaningless.
onSettingsChanged((_settings, keys) => {
  if (keys.includes('token')) pingCache = null;
});

function baseUrl(port) {
  return `http://127.0.0.1:${port}/api/v1`;
}

function headers(token, withBody) {
  const h = { 'X-GoDownload-Client': `extension/${extensionVersion}` };
  if (token) h.Authorization = `Bearer ${token}`;
  if (withBody) h['Content-Type'] = 'application/json';
  return h;
}

async function request(path, { method = 'GET', body, timeout, port, token, requireJson = false }) {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), timeout);
  try {
    const res = await fetch(`${baseUrl(port)}${path}`, {
      method,
      headers: headers(token, body !== undefined),
      body: body === undefined ? undefined : JSON.stringify(body),
      signal: controller.signal,
      cache: 'no-store',
      credentials: 'omit',   // never leak browser cookies for 127.0.0.1
      redirect: 'error',     // the app never redirects; refuse to follow anywhere else
    });
    let data = null;
    try {
      data = await res.json();
    } catch {
      // Non-JSON bodies are only acceptable for error statuses.
    }
    if (res.status === 401) throw new ApiError('unauthorized', 'Pairing token missing or invalid', 401);
    if (!res.ok) {
      const msg = (data && (data.error || data.message)) || `HTTP ${res.status}`;
      throw new ApiError('http', String(msg), res.status);
    }
    if (data === null || typeof data !== 'object') {
      // A 2xx without a JSON body still means the app accepted the request;
      // treating it as failure would make us restart the download in the browser
      // and produce a duplicate.
      if (requireJson) throw new ApiError('bad-response', 'Unexpected response from GoDownload', res.status);
      data = {};
    }
    return { status: res.status, data };
  } catch (err) {
    if (err instanceof ApiError) throw err;
    if (err?.name === 'AbortError') throw new ApiError('timeout', 'GoDownload did not respond in time');
    throw new ApiError('offline', 'GoDownload app is not running');
  } finally {
    clearTimeout(timer);
  }
}

async function connectionConfig(override) {
  const s = await getSettings();
  return { port: override?.port ?? foundPort ?? APP_PORTS[0], token: override?.token ?? s.token };
}

/** Pings one port; resolves to the app's info, or throws. */
async function pingPort(port, token) {
  const { data } = await request('/ping', { timeout: PING_TIMEOUT_MS, requireJson: true, port, token });
  if (data.app !== 'GoDownload') throw new ApiError('bad-response', 'Another application is using this port');
  return data;
}

/**
 * Looks for the app on its other ports, all at once. Resolves to the
 * first port that answers as GoDownload (or rejects with our token), else null.
 */
async function discover(skip, token) {
  const candidates = APP_PORTS.filter((p) => p !== skip);
  const attempts = candidates.map((port) =>
    pingPort(port, token).then(
      (info) => ({ port, info }),
      (err) => (err.code === 'unauthorized' ? { port, unauthorized: true } : Promise.reject(err)),
    ),
  );
  try {
    return await Promise.any(attempts);
  } catch {
    return null;
  }
}

/**
 * Checks whether the app is reachable.
 * @param {{force?: boolean, port?: number, token?: string}} [opts]
 *   `port`/`token` test unsaved values (options page) and bypass the cache.
 * @returns {Promise<{state: string, info: object|null, error?: string}>}
 */
export async function ping(opts = {}) {
  const isOverride = opts.port !== undefined || opts.token !== undefined;
  if (!opts.force && !isOverride && pingCache && Date.now() - pingCache.at < PING_CACHE_MS) {
    return pingCache;
  }
  if (!isOverride && pingInFlight) return pingInFlight;

  const run = (async () => {
    const cfg = await connectionConfig(isOverride ? opts : undefined);
    let result;
    try {
      const data = await pingPort(cfg.port, cfg.token);
      result = { state: Connection.CONNECTED, info: data, port: cfg.port };
    } catch (err) {
      const state = err.code === 'unauthorized' ? Connection.UNAUTHORIZED : Connection.OFFLINE;
      result = { state, info: null, error: err.message, port: cfg.port };
      if (state === Connection.OFFLINE && opts.port === undefined) {
        // The app may be on another of its ports (the first was taken or
        // reserved, or it restarted elsewhere); look before saying it isn't
        // running.
        const found = await discover(cfg.port, cfg.token);
        if (found) {
          foundPort = found.port;
          log.info(`GoDownload found on port ${found.port}`);
          result = found.unauthorized
            ? { state: Connection.UNAUTHORIZED, info: null, error: 'Pairing token missing or invalid', port: found.port }
            : { state: Connection.CONNECTED, info: found.info, port: found.port };
        } else {
          foundPort = null;
          result.error = "GoDownload isn't answering on this computer. Is the app open?";
        }
      }
    }
    result.at = Date.now();
    if (!isOverride) pingCache = result;
    return result;
  })();

  if (isOverride) return run;
  pingInFlight = run;
  try {
    return await run;
  } finally {
    pingInFlight = null;
  }
}

/** Records the outcome of a real request so the popup reflects it immediately. */
function noteOutcome(err) {
  if (!err) {
    if (pingCache) pingCache = { ...pingCache, state: Connection.CONNECTED, at: Date.now() };
    return;
  }
  if (err.code === 'unauthorized') pingCache = { state: Connection.UNAUTHORIZED, info: null, error: err.message, at: Date.now() };
  else if (err.code === 'offline' || err.code === 'timeout') pingCache = null;
}

async function post(path, body) {
  const cfg = await connectionConfig();
  try {
    const { data } = await request(path, { method: 'POST', body, timeout: POST_TIMEOUT_MS, ...cfg });
    noteOutcome(null);
    return data;
  } catch (err) {
    noteOutcome(err);
    log.warn(`POST ${path} failed:`, err.message);
    throw err;
  }
}

/**
 * Hands one download to the app.
 * @param {import('./payload.js').DownloadRequest} item
 * @returns {Promise<{id: string}>}
 */
export function sendDownload(item) {
  return post('/downloads', item);
}

/**
 * Hands a batch to the app, which shows its own picker dialog.
 * @param {{items: object[], pageUrl: string, pageTitle: string}} batch
 * @returns {Promise<{ids: string[]}>}
 */
export function sendBatch(batch) {
  return post('/downloads/batch', batch);
}
