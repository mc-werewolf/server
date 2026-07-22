type HealthStatus = { ok: boolean };

async function checkEndpoint(url: string): Promise<HealthStatus> {
  try {
    const response = await fetch(url, { cache: "no-store" });
    return { ok: response.ok };
  } catch {
    return { ok: false };
  }
}

const launcherDownload =
  "https://github.com/mc-werewolf/bds-launcher/releases/latest/download/bds-launcher-windows-x64-setup.exe";

export default async function Home() {
  const backendUrl = process.env.BACKEND_URL ?? "http://backend:8000";
  const status = await checkEndpoint(`${backendUrl}/api/health`);

  return (
    <main className="min-h-screen bg-zinc-950 px-6 py-16 text-zinc-50 sm:px-10">
      <div className="mx-auto flex min-h-[calc(100vh-8rem)] max-w-5xl flex-col justify-center">
        <div className="mb-8 flex items-center gap-3 text-sm text-zinc-400">
          <span className="h-2 w-2 rounded-full bg-red-500" />
          MC WEREWOLF
        </div>

        <div className="grid items-end gap-12 lg:grid-cols-[1.35fr_0.65fr]">
          <section>
            <p className="mb-4 font-mono text-sm uppercase tracking-[0.22em] text-red-400">
              Bedrock Dedicated Server Launcher
            </p>
            <h1 className="max-w-3xl text-5xl font-semibold leading-[1.05] tracking-tight sm:text-7xl">
              仲間とすぐに、<br />人狼ワールドへ。
            </h1>
            <p className="mt-7 max-w-2xl text-lg leading-8 text-zinc-400">
              BDS、Werewolf ワールド、Kairo アドオンをまとめて準備します。
              面倒なサーバー構築なしで、自分の PC からゲームを開始できます。
            </p>
            <div className="mt-10 flex flex-wrap items-center gap-4">
              <a
                href={launcherDownload}
                className="rounded-xl bg-red-600 px-7 py-4 font-semibold text-white transition hover:bg-red-500"
              >
                Windows 版をダウンロード
              </a>
              <span className="text-sm text-zinc-500">Windows 10 / 11 · x64</span>
            </div>
          </section>

          <aside className="rounded-2xl border border-white/10 bg-white/[0.04] p-6">
            <h2 className="text-sm font-medium text-zinc-300">ランチャーが自動で行うこと</h2>
            <ol className="mt-5 space-y-4 text-sm text-zinc-400">
              <li><span className="mr-3 font-mono text-red-400">01</span>BDS のセットアップ</li>
              <li><span className="mr-3 font-mono text-red-400">02</span>最新ワールドの取得</li>
              <li><span className="mr-3 font-mono text-red-400">03</span>Kairo アドオンの適用</li>
              <li><span className="mr-3 font-mono text-red-400">04</span>公開接続とサーバー起動</li>
            </ol>
            <div className="mt-7 flex items-center gap-2 border-t border-white/10 pt-5 text-xs text-zinc-500">
              <span className={`h-2 w-2 rounded-full ${status.ok ? "bg-emerald-500" : "bg-amber-500"}`} />
              中心サーバー {status.ok ? "稼働中" : "確認中"}
            </div>
          </aside>
        </div>
      </div>
    </main>
  );
}
