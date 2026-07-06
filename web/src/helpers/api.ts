const STARTUP_RETRY_WINDOW_MS = 8000;
const STARTUP_RETRY_INTERVAL_MS = 150;

function sleep(ms: number) {
  return new Promise((resolve) => window.setTimeout(resolve, ms));
}

function isLocalApiBase(apiBase: string) {
  try {
    const url = new URL(apiBase);
    return url.hostname === '127.0.0.1' || url.hostname === 'localhost';
  } catch {
    return false;
  }
}

async function fetchWithStartupRetry(apiBase: string, path: string, init: RequestInit) {
  const url = `${apiBase}${path}`;
  if (!isLocalApiBase(apiBase)) {
    return fetch(url, init);
  }

  const deadline = Date.now() + STARTUP_RETRY_WINDOW_MS;
  let lastError: unknown;

  while (Date.now() <= deadline) {
    try {
      return await fetch(url, init);
    } catch (error) {
      lastError = error;
      if (Date.now() >= deadline) {
        break;
      }
      await sleep(STARTUP_RETRY_INTERVAL_MS);
    }
  }

  throw lastError;
}

/** Thin wrapper around fetch. Retries briefly while the packaged local API finishes booting. */
export async function request<T>(
  apiBase: string,
  token: string,
  path: string,
  payload?: unknown,
): Promise<T> {
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  };
  if (token.trim().length > 0) {
    headers.Authorization = `Bearer ${token.trim()}`;
  }

  let response: Response;
  try {
    response = await fetchWithStartupRetry(apiBase, path, {
      method: payload === undefined ? 'GET' : 'POST',
      headers,
      body: payload === undefined ? undefined : JSON.stringify(payload),
    });
  } catch (networkError) {
    throw new Error(`无法连接后端服务 (${apiBase})：${networkError instanceof Error ? networkError.message : String(networkError)}`);
  }

  const body = await response.json().catch(() => ({}));
  if (!response.ok) {
    const errorMessage = typeof body?.error === 'string' ? body.error : `HTTP ${response.status}`;
    throw new Error(errorMessage);
  }
  return body as T;
}
