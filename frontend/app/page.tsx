type HealthStatus = {
  ok: boolean;
  body: string;
};

async function checkEndpoint(url: string): Promise<HealthStatus> {
  try {
    const res = await fetch(url, { cache: "no-store" });
    const body = await res.text();
    return { ok: res.ok, body };
  } catch (err) {
    return { ok: false, body: err instanceof Error ? err.message : String(err) };
  }
}

export default async function Home() {
  const backendUrl = process.env.BACKEND_URL ?? "http://backend:8000";
  const [api, db] = await Promise.all([
    checkEndpoint(`${backendUrl}/api/health`),
    checkEndpoint(`${backendUrl}/api/health/db`),
  ]);

  return (
    <div className="flex min-h-screen flex-col items-center justify-center gap-8 bg-zinc-50 p-16 font-sans dark:bg-black">
      <h1 className="text-3xl font-semibold text-black dark:text-zinc-50">
        Werewolf Server
      </h1>
      <div className="flex w-full max-w-md flex-col gap-4">
        <StatusCard label="API" status={api} />
        <StatusCard label="Database" status={db} />
      </div>
    </div>
  );
}

function StatusCard({ label, status }: { label: string; status: HealthStatus }) {
  return (
    <div className="flex items-center justify-between rounded-lg border border-black/10 bg-white px-6 py-4 dark:border-white/10 dark:bg-zinc-900">
      <span className="font-medium text-black dark:text-zinc-50">{label}</span>
      <div className="flex items-center gap-2">
        <span
          className={`h-2.5 w-2.5 rounded-full ${status.ok ? "bg-green-500" : "bg-red-500"}`}
        />
        <span className="font-mono text-sm text-zinc-600 dark:text-zinc-400">
          {status.body}
        </span>
      </div>
    </div>
  );
}
