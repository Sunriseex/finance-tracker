import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { api, clearStoredSession, setStoredToken } from "./client";

const session = {
  user: { id: "user-1", email: "user@example.com", primary_currency: "RUB" },
  access_token: "new-access",
  access_expires_at: "2026-05-11T10:00:00Z",
};

describe("api client", () => {
  beforeEach(() => {
    localStorage.clear();
    vi.stubGlobal("fetch", vi.fn());
    vi.stubGlobal("crypto", { randomUUID: () => "idem-key" });
  });

  afterEach(() => {
    clearStoredSession();
    vi.unstubAllGlobals();
  });

  it("refreshes once after unauthorized API response", async () => {
    setStoredToken("old-access");
    localStorage.setItem("capitalflow_refresh_token", "legacy-refresh");
    const fetchMock = vi.mocked(fetch);
    fetchMock
      .mockResolvedValueOnce(jsonResponse({ error: { code: "unauthorized", message: "Unauthorized" } }, 401))
      .mockResolvedValueOnce(jsonResponse(session))
      .mockResolvedValueOnce(jsonResponse({ user: session.user }));

    await expect(api.profile()).resolves.toEqual({ user: session.user });

    expect(fetchMock).toHaveBeenNthCalledWith(1, "/api/v1/settings/profile", expect.objectContaining({
      headers: expect.any(Headers),
    }));
    expect(fetchMock).toHaveBeenNthCalledWith(2, "/auth/refresh", expect.any(Object));
    expect(fetchMock).toHaveBeenNthCalledWith(3, "/api/v1/settings/profile", expect.any(Object));

    const refreshInit = fetchMock.mock.calls[1]?.[1];
    expect(refreshInit).toEqual(expect.objectContaining({
      method: "POST",
      credentials: "include",
    }));
    expect(refreshInit?.body).toBeUndefined();
    expect(localStorage.getItem("capitalflow_refresh_token")).toBeNull();
  });

  it("does not store refresh tokens from auth responses", async () => {
    const fetchMock = vi.mocked(fetch);
    fetchMock
      .mockResolvedValueOnce(jsonResponse({
        ...session,
        refresh_token: "login-refresh",
        refresh_expires_at: "2026-06-11T10:00:00Z",
      }))
      .mockResolvedValueOnce(jsonResponse({
        ...session,
        access_token: "setup-access",
        refresh_token: "setup-refresh",
        refresh_expires_at: "2026-06-11T10:00:00Z",
      }));

    await expect(api.login({ email: "user@example.com", password: "password" })).resolves.toMatchObject({
      access_token: "new-access",
    });
    expect(localStorage.getItem("capitalflow_api_token")).toBe("new-access");
    expect(localStorage.getItem("capitalflow_refresh_token")).toBeNull();

    await expect(api.setup({
      email: "setup@example.com",
      password: "password",
      primary_currency: "RUB",
    })).resolves.toMatchObject({
      access_token: "setup-access",
    });
    expect(localStorage.getItem("capitalflow_api_token")).toBe("setup-access");
    expect(localStorage.getItem("capitalflow_refresh_token")).toBeNull();

    for (const call of fetchMock.mock.calls) {
      expect(call[1]).toEqual(expect.objectContaining({
        method: "POST",
        credentials: "include",
      }));
    }
  });

  it("logs out using cookies only", async () => {
    setStoredToken("access");
    localStorage.setItem("capitalflow_refresh_token", "legacy-refresh");
    vi.mocked(fetch).mockResolvedValueOnce(new Response(null, { status: 204 }));

    await api.logout();

    const init = vi.mocked(fetch).mock.calls[0]?.[1];
    expect(vi.mocked(fetch)).toHaveBeenCalledWith("/auth/logout", expect.objectContaining({
      method: "POST",
      credentials: "include",
    }));
    expect(init?.body).toBeUndefined();
    expect(localStorage.getItem("capitalflow_api_token")).toBeNull();
    expect(localStorage.getItem("capitalflow_refresh_token")).toBeNull();
  });

  it("adds idempotency keys to mutations", async () => {
    setStoredToken("access");
    vi.mocked(fetch).mockResolvedValueOnce(jsonResponse({
      id: "account-1",
      name: "Card",
      type: "card",
      currency: "RUB",
      is_active: true,
      opened_at: "2026-05-11",
      created_at: "2026-05-11T00:00:00Z",
      updated_at: "2026-05-11T00:00:00Z",
    }));

    await api.createAccount({ name: "Card", bank: "", type: "card", currency: "RUB", opened_at: "2026-05-11" });

    const init = vi.mocked(fetch).mock.calls[0]?.[1];
    const headers = init?.headers as Headers;
    expect(headers.get("Idempotency-Key")).toBe("idem-key");
  });
});

function jsonResponse(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}
