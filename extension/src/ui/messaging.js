/**
 * Request/response helper for extension pages talking to the background.
 */
import { api } from '../lib/browser.js';

/**
 * @param {string} type  handler name, e.g. "popup:getState"
 * @param {object} [payload]
 * @returns {Promise<any>} the handler's data; rejects with its error message
 */
export async function callBackground(type, payload = {}) {
  let response;
  try {
    response = await api.runtime.sendMessage({ type, payload });
  } catch (err) {
    throw new Error(`Extension background is not responding (${err?.message ?? err}).`);
  }
  if (!response) throw new Error('No response from the extension background.');
  if (!response.ok) {
    const error = new Error(response.error || 'Unknown error');
    error.code = response.code;
    throw error;
  }
  return response.data;
}
