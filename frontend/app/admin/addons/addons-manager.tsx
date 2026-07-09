"use client";

import { useState } from "react";
import type { Addon, AddonProperties } from "./types";

function formatVersion(properties: AddonProperties | null): string | null {
  const v = properties?.header?.version;
  if (!v) return null;
  const base = `${v.major ?? 0}.${v.minor ?? 0}.${v.patch ?? 0}`;
  return v.prerelease ? `${base}-${v.prerelease}` : base;
}

async function fetchAddons(): Promise<Addon[]> {
  const res = await fetch("/api/admin/addons");
  if (!res.ok) throw new Error(`一覧の取得に失敗しました (${res.status})`);
  const data = await res.json();
  return data.addons ?? [];
}

export default function AddonsManager({ initialAddons }: { initialAddons: Addon[] }) {
  const [url, setUrl] = useState("");
  const [addons, setAddons] = useState<Addon[]>(initialAddons);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [busyId, setBusyId] = useState<string | null>(null);

  async function handleRegister(e: React.FormEvent) {
    e.preventDefault();
    if (!url.trim()) return;

    setLoading(true);
    setError(null);
    try {
      const res = await fetch("/api/admin/addons", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ url }),
      });
      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        throw new Error(body.error ?? `登録に失敗しました (${res.status})`);
      }
      setUrl("");
      setAddons(await fetchAddons());
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }

  async function handleRefresh(id: string) {
    setBusyId(id);
    setError(null);
    try {
      const res = await fetch(`/api/admin/addons/${id}/refresh`, { method: "POST" });
      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        throw new Error(body.error ?? `再取得に失敗しました (${res.status})`);
      }
      setAddons(await fetchAddons());
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusyId(null);
    }
  }

  return (
    <div className="flex min-h-screen flex-col items-center gap-8 bg-zinc-50 p-16 font-sans dark:bg-black">
      <h1 className="text-3xl font-semibold text-black dark:text-zinc-50">
        アドオン管理
      </h1>

      <form onSubmit={handleRegister} className="flex w-full max-w-2xl gap-2">
        <input
          type="text"
          value={url}
          onChange={(e) => setUrl(e.target.value)}
          placeholder="https://github.com/owner/repo"
          className="flex-1 rounded-lg border border-black/10 bg-white px-4 py-2 text-black dark:border-white/10 dark:bg-zinc-900 dark:text-zinc-50"
        />
        <button
          type="submit"
          disabled={loading}
          className="rounded-lg border border-black/10 bg-white px-4 py-2 font-medium text-black hover:bg-zinc-100 disabled:opacity-50 dark:border-white/10 dark:bg-zinc-900 dark:text-zinc-50 dark:hover:bg-zinc-800"
        >
          {loading ? "登録中..." : "登録"}
        </button>
      </form>

      {error && (
        <p className="w-full max-w-2xl text-sm text-red-600 dark:text-red-400">{error}</p>
      )}

      <div className="flex w-full max-w-2xl flex-col gap-6">
        {addons.length === 0 && (
          <p className="text-center text-sm text-zinc-500 dark:text-zinc-400">
            登録済みのアドオンはありません
          </p>
        )}

        {addons.map((addon) => (
          <div
            key={addon.id}
            className="flex flex-col gap-3 rounded-lg border border-black/10 bg-white p-6 dark:border-white/10 dark:bg-zinc-900"
          >
            <div className="flex items-center justify-between">
              <span className="font-mono font-medium text-black dark:text-zinc-50">
                {addon.github_owner}/{addon.github_repo}
              </span>
              <button
                type="button"
                onClick={() => handleRefresh(addon.id)}
                disabled={busyId === addon.id}
                className="rounded border border-black/10 px-3 py-1 text-sm text-black hover:bg-zinc-100 disabled:opacity-50 dark:border-white/10 dark:text-zinc-50 dark:hover:bg-zinc-800"
              >
                {busyId === addon.id ? "再取得中..." : "再取得"}
              </button>
            </div>

            <div className="flex flex-col gap-2">
              {addon.versions.length === 0 && (
                <p className="text-sm text-zinc-500 dark:text-zinc-400">バージョンなし</p>
              )}
              {addon.versions.map((v) => (
                <div
                  key={v.id}
                  className="flex items-center justify-between rounded border border-black/10 px-3 py-2 text-sm dark:border-white/10"
                >
                  <div className="flex flex-col">
                    <span className="font-mono text-black dark:text-zinc-50">
                      {v.tag_name}
                      {v.properties && (
                        <span className="ml-2 text-zinc-500 dark:text-zinc-400">
                          {v.properties.header?.name} v{formatVersion(v.properties)}
                        </span>
                      )}
                    </span>
                    {v.properties?.header?.description && (
                      <span className="text-zinc-500 dark:text-zinc-400">
                        {v.properties.header.description}
                      </span>
                    )}
                    {v.properties_error && (
                      <span className="text-red-600 dark:text-red-400">
                        properties.js not found: {v.properties_error}
                      </span>
                    )}
                  </div>
                  <a
                    href={`/api/addons/${addon.github_owner}/${addon.github_repo}/versions/${v.tag_name}/download`}
                    className="rounded border border-black/10 px-3 py-1 text-black hover:bg-zinc-100 dark:border-white/10 dark:text-zinc-50 dark:hover:bg-zinc-800"
                  >
                    DL
                  </a>
                </div>
              ))}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
