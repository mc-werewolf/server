import Link from "next/link";

export default function AdminPage() {
  return (
    <div className="flex min-h-screen flex-col items-center justify-center gap-8 bg-zinc-50 p-16 font-sans dark:bg-black">
      <h1 className="text-3xl font-semibold text-black dark:text-zinc-50">
        Admin
      </h1>
      <Link
        href="/admin/addons"
        className="rounded-lg border border-black/10 bg-white px-6 py-4 font-medium text-black hover:bg-zinc-100 dark:border-white/10 dark:bg-zinc-900 dark:text-zinc-50 dark:hover:bg-zinc-800"
      >
        アドオン管理
      </Link>
    </div>
  );
}
