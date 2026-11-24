/**
 * Prefixed console logging plus a helper that turns any promise into one that
 * never rejects, so no code path can produce an unhandled rejection.
 */
const PREFIX = '[GoDownload]';

export const log = {
  debug: (...args) => console.debug(PREFIX, ...args),
  info: (...args) => console.info(PREFIX, ...args),
  warn: (...args) => console.warn(PREFIX, ...args),
  error: (...args) => console.error(PREFIX, ...args),
};

/**
 * Runs `fn` (sync or async) and logs instead of throwing.
 * @template T
 * @param {string} label  context shown in the log line
 * @param {() => T | Promise<T>} fn
 * @param {T} [fallback]  value returned when `fn` fails
 * @returns {Promise<T>}
 */
export async function safely(label, fn, fallback = undefined) {
  try {
    return await fn();
  } catch (err) {
    log.error(`${label}:`, err);
    return fallback;
  }
}

