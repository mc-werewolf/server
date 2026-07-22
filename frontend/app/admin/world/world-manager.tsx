"use client";

import Link from "next/link";
import { useState } from "react";
import type { WorldRelease } from "./types";

function formatBytes(bytes: number): string {
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
}

export default function WorldManager({ initialWorld }: { initialWorld: WorldRelease | null }) {
  const [world, setWorld] = useState(initialWorld);
  const [version, setVersion] = useState(initialWorld?.version ?? "1.0.0");
  const [file, setFile] = useState<File | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function upload(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!file) return;
    setBusy(true);
    setError(null);
    try {
      const form = new FormData();
      form.set("version", version);
      form.set("world", file);
      const response = await fetch("/api/admin/world", { method: "POST", body: form });
      const body = await response.json().catch(() => ({}));
      if (!response.ok) throw new Error(body.error ?? `アップロードに失敗しました (${response.status})`);
      setWorld(body);
      setFile(null);
      event.currentTarget.reset();
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : String(cause));
    } finally {
      setBusy(false);
    }
  }

  return (
    <main className="min-h-screen bg-zinc-50 px-6 py-12 text-zinc-950 dark:bg-black dark:text-zinc-50">
      <div className="mx-auto max-w-2xl">
        <Link href="/admin" className="text-sm text-zinc-500 hover:text-zinc-800 dark:hover:text-zinc-200">← 管理画面</Link>
        <h1 className="mt-6 text-3xl font-semibold">Werewolf ワールド管理</h1>
        <p className="mt-2 text-sm leading-6 text-zinc-500">
          Minecraft からエクスポートした .mcworld を登録します。ランチャーは初回構築時に最新ワールドを取得します。
        </p>

        <section className="mt-8 rounded-xl border border-black/10 bg-white p-6 dark:border-white/10 dark:bg-zinc-900">
          <h2 className="font-medium">現在のワールド</h2>
          {world ? (
            <dl className="mt-4 grid grid-cols-[8rem_1fr] gap-y-2 text-sm">
              <dt className="text-zinc-500">バージョン</dt><dd>{world.version}</dd>
              <dt className="text-zinc-500">ファイル</dt><dd>{world.fileName}</dd>
              <dt className="text-zinc-500">サイズ</dt><dd>{formatBytes(world.fileSize)}</dd>
              <dt className="text-zinc-500">更新日時</dt><dd>{new Date(world.updatedAt).toLocaleString("ja-JP")}</dd>
              <dt className="text-zinc-500">SHA-256</dt><dd className="truncate font-mono text-xs" title={world.sha256}>{world.sha256}</dd>
              <dt /><dd><a href={world.downloadUrl} className="text-red-600 hover:underline">登録ファイルを確認</a></dd>
            </dl>
          ) : (
            <p className="mt-4 text-sm text-zinc-500">まだワールドが登録されていません。</p>
          )}
        </section>

        <form onSubmit={upload} className="mt-6 space-y-5 rounded-xl border border-black/10 bg-white p-6 dark:border-white/10 dark:bg-zinc-900">
          <h2 className="font-medium">新しいワールドを公開</h2>
          <label className="block text-sm">
            <span className="mb-2 block text-zinc-500">バージョン</span>
            <input required value={version} onChange={(event) => setVersion(event.target.value)} pattern="[A-Za-z0-9][A-Za-z0-9._-]{0,63}" className="w-full rounded-lg border border-black/10 bg-transparent px-4 py-3 dark:border-white/10" />
          </label>
          <label className="block text-sm">
            <span className="mb-2 block text-zinc-500">ワールドファイル</span>
            <input required type="file" accept=".mcworld,.zip" onChange={(event) => setFile(event.target.files?.[0] ?? null)} className="w-full rounded-lg border border-dashed border-black/20 p-4 file:mr-4 file:rounded file:border-0 file:px-3 file:py-2 dark:border-white/20" />
          </label>
          {error && <p className="text-sm text-red-600 dark:text-red-400">{error}</p>}
          <button disabled={busy || !file} className="rounded-lg bg-red-600 px-5 py-3 font-medium text-white hover:bg-red-500 disabled:opacity-50">
            {busy ? "アップロード中..." : "このワールドを公開"}
          </button>
        </form>
      </div>
    </main>
  );
}
