import AddonsManager from "./addons-manager";
import type { Addon } from "./types";

async function fetchInitialAddons(): Promise<Addon[]> {
  const backendUrl = process.env.BACKEND_URL ?? "http://backend:8000";
  try {
    const res = await fetch(`${backendUrl}/api/admin/addons`, { cache: "no-store" });
    if (!res.ok) return [];
    const data = await res.json();
    return data.addons ?? [];
  } catch {
    return [];
  }
}

export default async function AdminAddonsPage() {
  const initialAddons = await fetchInitialAddons();
  return <AddonsManager initialAddons={initialAddons} />;
}
