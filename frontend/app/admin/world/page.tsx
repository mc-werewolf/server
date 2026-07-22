import WorldManager from "./world-manager";
import type { WorldRelease } from "./types";

async function fetchCurrentWorld(): Promise<WorldRelease | null> {
  const backendUrl = process.env.BACKEND_URL ?? "http://backend:8000";
  try {
    const response = await fetch(`${backendUrl}/api/admin/world`, { cache: "no-store" });
    if (!response.ok) return null;
    return await response.json();
  } catch {
    return null;
  }
}

export default async function AdminWorldPage() {
  return <WorldManager initialWorld={await fetchCurrentWorld()} />;
}
