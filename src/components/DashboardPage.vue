<script setup lang="ts">
import type { URLItem } from "../types";

defineProps<{
  urls: URLItem[];
  activeUrls: URLItem[];
  disabledUrls: URLItem[];
  urlsLoading: boolean;
  originalUrl: string;
  shortUrl: string;
  copied: boolean;
  urlError: string;
  urlMessage: string;
  urlLoading: boolean;
}>();

const emit = defineEmits<{
  "update:originalUrl": [value: string];
  create: [];
  copy: [value: string];
  navigate: [page: "urls"];
}>();
</script>

<template>
  <section>
    <div class="mb-8 flex flex-col justify-between gap-4 sm:flex-row sm:items-end"><div><p class="mb-2 text-sm font-medium text-slate-500">Selamat datang kembali</p><h1 class="text-3xl font-semibold tracking-tight sm:text-4xl">Dashboard.</h1><p class="mt-3 text-slate-600">Ringkasan workspace dan akses cepat untuk membuat tautan.</p></div><button type="button" class="rounded-md bg-slate-900 px-4 py-2.5 text-sm font-medium text-white hover:bg-slate-700" @click="emit('navigate', 'urls')">Lihat semua URL</button></div>
    <div class="mb-8 grid gap-4 sm:grid-cols-3">
      <div class="rounded-xl border border-stone-300 bg-white p-5"><p class="text-sm text-slate-500">Total URL</p><p class="mt-3 text-3xl font-semibold">{{ urls.length }}</p><p class="mt-2 text-xs text-slate-400">Semua tautan di workspace</p></div>
      <div class="rounded-xl border border-stone-300 bg-white p-5"><p class="text-sm text-slate-500">URL aktif</p><p class="mt-3 text-3xl font-semibold text-green-700">{{ activeUrls.length }}</p><p class="mt-2 text-xs text-slate-400">Siap digunakan</p></div>
      <div class="rounded-xl border border-stone-300 bg-white p-5"><p class="text-sm text-slate-500">Dinonaktifkan</p><p class="mt-3 text-3xl font-semibold text-slate-400">{{ disabledUrls.length }}</p><p class="mt-2 text-xs text-slate-400">Tidak menerima redirect</p></div>
    </div>
    <div class="grid gap-6 xl:grid-cols-[1.05fr_0.95fr]">
      <section class="rounded-xl border border-stone-300 bg-white p-5 shadow-sm sm:p-7">
        <div class="mb-6"><p class="text-sm font-medium text-slate-500">Quick action</p><h2 class="mt-1 text-xl font-semibold tracking-tight">Buat short URL</h2><p class="mt-2 text-sm text-slate-500">Masukkan URL tujuan dan bagikan versi ringkasnya.</p></div>
        <div v-if="urlError" class="mb-5 rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700" role="alert">{{ urlError }}</div>
        <div v-if="shortUrl" class="mb-6 rounded-md border border-green-200 bg-green-50 p-4" role="status"><p class="text-sm font-medium text-green-800">{{ urlMessage }}</p><div class="mt-3 flex gap-2"><input :value="shortUrl" class="min-w-0 flex-1 rounded-md border border-green-200 bg-white px-3 py-2.5 text-sm text-green-900 outline-none" readonly><button type="button" class="rounded-md bg-slate-900 px-3 py-2 text-sm font-medium text-white" @click="emit('copy', shortUrl)">{{ copied ? "Tersalin" : "Salin" }}</button></div></div>
        <form class="space-y-4" @submit.prevent="emit('create')"><div><label for="original_url" class="mb-1.5 block text-sm font-medium text-slate-700">URL tujuan</label><input id="original_url" :value="originalUrl" type="url" autocomplete="url" required maxlength="2048" placeholder="https://contoh.com/halaman-panjang" class="w-full rounded-md border border-stone-300 bg-stone-50 px-4 py-3 outline-none focus:border-slate-500 focus:ring-1 focus:ring-slate-500" @input="emit('update:originalUrl', ($event.target as HTMLInputElement).value)"></div><button type="submit" :disabled="urlLoading" class="rounded-md bg-slate-900 px-5 py-2.5 text-sm font-medium text-white hover:bg-slate-700 disabled:opacity-60">{{ urlLoading ? "Membuat tautan..." : "Buat short URL" }}</button></form>
      </section>
      <section class="rounded-xl border border-stone-300 bg-white p-5 shadow-sm sm:p-7">
        <div class="mb-5 flex items-end justify-between"><div><p class="text-sm font-medium text-slate-500">Aktivitas terbaru</p><h2 class="mt-1 text-xl font-semibold tracking-tight">URL terbaru</h2></div><button type="button" class="text-sm font-medium text-slate-500 hover:text-slate-900" @click="emit('navigate', 'urls')">Kelola semua</button></div>
        <div v-if="urlsLoading" class="py-8 text-center text-sm text-slate-500">Memuat aktivitas...</div>
        <div v-else-if="urls.length === 0" class="rounded-lg bg-stone-50 px-4 py-8 text-center text-sm text-slate-500">Belum ada aktivitas. Buat URL pertamamu.</div>
        <ul v-else class="divide-y divide-stone-200">
          <li v-for="item in urls.slice(0, 5)" :key="item.id" class="py-3">
            <div class="flex items-center justify-between gap-3"><div class="flex min-w-0 items-center gap-2"><p class="truncate text-sm font-medium text-slate-900">{{ item.short_url }}</p><a :href="`/qr/${item.id}.png`" download class="shrink-0 text-xs text-slate-500 hover:text-slate-900">QR</a></div><span class="shrink-0 text-[10px] font-semibold uppercase tracking-wider" :class="item.status === 'active' ? 'text-green-700' : 'text-slate-400'">{{ item.status }}</span></div>
            <div class="mt-1 flex items-center justify-between gap-3"><p class="truncate text-xs text-slate-500">{{ item.original_url }}</p><p class="shrink-0 text-xs text-slate-400">{{ item.total_clicks }} klik</p></div>
          </li>
        </ul>
      </section>
    </div>
  </section>
</template>
