<script setup lang="ts">
import type { Page } from "../types";

defineProps<{ page: Page; mobileMenuOpen: boolean; role: string }>();
const emit = defineEmits<{ navigate: [page: Page]; toggleMenu: []; logout: [] }>();
const titles: Record<Page, string> = { dashboard: "Dashboard", urls: "URL List", "url-detail": "URL Detail", users: "User Management", audit: "Audit Log" };
</script>

<template>
  <header class="relative flex h-16 items-center justify-between border-b border-stone-300 bg-white px-5 sm:px-8">
    <div class="flex items-center gap-3"><button type="button" class="rounded-md border border-stone-300 px-2 py-1.5 text-sm lg:hidden" @click="emit('toggleMenu')">Menu</button><div><p class="text-xs text-slate-400">Workspace</p><p class="font-medium">{{ titles[page] }}</p></div></div>
    <button type="button" class="text-sm font-medium text-slate-600 hover:text-slate-950" @click="emit('logout')">Keluar</button>
    <nav v-if="mobileMenuOpen" class="absolute left-4 right-4 top-14 z-10 rounded-lg border border-stone-300 bg-white p-2 shadow-lg lg:hidden">
      <button type="button" class="block w-full rounded px-3 py-2 text-left text-sm hover:bg-stone-100" @click="emit('navigate', 'dashboard')">Dashboard</button>
      <button type="button" class="block w-full rounded px-3 py-2 text-left text-sm hover:bg-stone-100" @click="emit('navigate', 'urls')">URL List</button>
      <button v-if="role === 'admin'" type="button" class="block w-full rounded px-3 py-2 text-left text-sm hover:bg-stone-100" @click="emit('navigate', 'users')">User Management</button>
      <button v-if="role === 'admin'" type="button" class="block w-full rounded px-3 py-2 text-left text-sm hover:bg-stone-100" @click="emit('navigate', 'audit')">Audit Log</button>
    </nav>
  </header>
</template>
