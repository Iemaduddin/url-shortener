<script setup lang="ts">
import type { AuditLog } from "../types";

defineProps<{ logs: AuditLog[]; loading: boolean; error: string; page: number; totalPages: number }>();
const emit = defineEmits<{ "update:page": [value: number] }>();
</script>

<template>
  <section>
    <div class="mb-8"><p class="mb-2 text-sm font-medium text-slate-500">Administration</p><h1 class="text-3xl font-semibold tracking-tight sm:text-4xl">Audit Log.</h1><p class="mt-3 text-slate-600">Riwayat tindakan administratif pada aplikasi.</p></div>
    <section class="overflow-hidden rounded-xl border border-stone-300 bg-white shadow-sm">
      <div v-if="error" class="m-4 rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">{{ error }}</div>
      <div v-else-if="loading" class="p-8 text-center text-sm text-slate-500">Memuat audit log...</div>
      <div v-else-if="logs.length === 0" class="p-10 text-center text-sm text-slate-500">Belum ada aktivitas administratif.</div>
      <div v-else class="overflow-x-auto"><table class="w-full min-w-[760px] text-left text-sm"><thead class="border-b border-stone-200 bg-stone-50 text-xs uppercase tracking-wide text-slate-500"><tr><th class="px-5 py-3 font-medium">Waktu</th><th class="px-5 py-3 font-medium">Aksi</th><th class="px-5 py-3 font-medium">Resource</th><th class="px-5 py-3 font-medium">Actor</th><th class="px-5 py-3 font-medium">Metadata</th></tr></thead><tbody class="divide-y divide-stone-200"><tr v-for="log in logs" :key="log.id"><td class="whitespace-nowrap px-5 py-4 text-slate-500">{{ new Date(log.created_at).toLocaleString("id-ID") }}</td><td class="px-5 py-4 font-medium text-slate-900">{{ log.action }}</td><td class="px-5 py-4 text-slate-600">{{ log.resource_type }}<span v-if="log.resource_id" class="block text-xs text-slate-400">{{ log.resource_id }}</span></td><td class="px-5 py-4 text-slate-600">{{ log.actor_user_id || "system" }}</td><td class="max-w-xs truncate px-5 py-4 text-xs text-slate-500">{{ log.metadata }}</td></tr></tbody></table></div>
      <div v-if="totalPages > 1" class="flex items-center justify-between border-t border-stone-200 p-4"><button type="button" class="rounded-md border border-stone-300 px-3 py-2 text-sm disabled:opacity-40" :disabled="page === 1" @click="emit('update:page', page - 1)">Sebelumnya</button><span class="text-sm text-slate-500">Halaman {{ page }} dari {{ totalPages }}</span><button type="button" class="rounded-md border border-stone-300 px-3 py-2 text-sm disabled:opacity-40" :disabled="page === totalPages" @click="emit('update:page', page + 1)">Berikutnya</button></div>
    </section>
  </section>
</template>
