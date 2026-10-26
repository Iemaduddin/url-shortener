<script setup lang="ts">
import type { Page } from "../types";

defineProps<{ role: string; currentPage: Page }>();
const emit = defineEmits<{ navigate: [page: Page] }>();
</script>

<template>
  <aside class="hidden w-64 shrink-0 border-r border-stone-300 bg-white p-5 lg:flex lg:flex-col">
    <div class="mb-10 flex items-center justify-between"><p class="text-lg font-semibold tracking-tight">shortly.</p><span class="rounded bg-stone-100 px-2 py-1 text-[10px] font-semibold uppercase tracking-wider text-slate-500">App</span></div>
    <p class="mb-3 px-3 text-[11px] font-semibold uppercase tracking-widest text-slate-400">Workspace</p>
    <nav class="space-y-1 text-sm">
      <button type="button" :class="currentPage === 'dashboard' ? 'bg-slate-900 font-medium text-white' : 'text-slate-500 hover:bg-stone-100 hover:text-slate-900'" class="flex w-full rounded-md px-3 py-2.5 text-left" @click="emit('navigate', 'dashboard')">Dashboard</button>
      <button type="button" :class="currentPage === 'urls' ? 'bg-slate-900 font-medium text-white' : 'text-slate-500 hover:bg-stone-100 hover:text-slate-900'" class="flex w-full rounded-md px-3 py-2.5 text-left" @click="emit('navigate', 'urls')">URL List</button>
      <button v-if="role === 'admin'" type="button" :class="currentPage === 'users' ? 'bg-slate-900 font-medium text-white' : 'text-slate-500 hover:bg-stone-100 hover:text-slate-900'" class="flex w-full rounded-md px-3 py-2.5 text-left" @click="emit('navigate', 'users')">User Management</button>
      <button v-if="role === 'admin'" type="button" :class="currentPage === 'audit' ? 'bg-slate-900 font-medium text-white' : 'text-slate-500 hover:bg-stone-100 hover:text-slate-900'" class="flex w-full rounded-md px-3 py-2.5 text-left" @click="emit('navigate', 'audit')">Audit Log</button>
    </nav>
    <div class="mt-auto border-t border-stone-200 pt-5 text-xs text-slate-500"><p>Signed in as</p><p class="mt-1 font-medium text-slate-900">{{ role }}</p></div>
  </aside>
</template>
