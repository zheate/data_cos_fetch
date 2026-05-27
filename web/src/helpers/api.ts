/** Thin wrapper around fetch – keeps the same contract as the original request(). */
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
    response = await fetch(`${apiBase}${path}`, {
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
