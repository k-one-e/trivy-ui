import { afterEach, describe, expect, it, vi } from "vitest"

async function loadClient() {
  vi.resetModules()
  const fakeWindow = new EventTarget()
  vi.stubGlobal("window", fakeWindow)
  const client = await import("./client")
  return { client, fakeWindow }
}

describe("fetchApi when the page is left", () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it("reports a request cancelled by navigation as aborted", async () => {
    const { client, fakeWindow } = await loadClient()
    vi.stubGlobal("fetch", () => {
      fakeWindow.dispatchEvent(new Event("beforeunload"))
      return Promise.reject(new TypeError("NetworkError when attempting to fetch resource."))
    })
    await expect(client.api.getClusters()).rejects.toMatchObject({ name: "AbortError" })
  })

  it("still reports network failures while the page stays open", async () => {
    const { client } = await loadClient()
    vi.stubGlobal("fetch", () => Promise.reject(new TypeError("Failed to fetch")))
    await expect(client.api.getClusters()).rejects.toBeInstanceOf(TypeError)
  })

  it("treats requests started after the page is shown again normally", async () => {
    const { client, fakeWindow } = await loadClient()
    fakeWindow.dispatchEvent(new Event("pagehide"))
    await new Promise((resolve) => setTimeout(resolve, 2))
    vi.stubGlobal("fetch", () => Promise.reject(new TypeError("Failed to fetch")))
    await expect(client.api.getClusters()).rejects.toBeInstanceOf(TypeError)
  })
})
