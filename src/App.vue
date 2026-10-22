<script setup lang="ts">
import { ref } from "vue";

interface CreateURLResponse {
  short_url?: string;
  error?: { message?: string };
}

const originalUrl = ref<string>("");
const shortUrl = ref<string>("");
const error = ref<string>("");
const copied = ref(false);
const loading = ref(false);

async function createShortUrl() {
  error.value = "";
  shortUrl.value = "";
  copied.value = false;
  loading.value = true;

  try {
    const response = await fetch("/api/urls", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ original_url: originalUrl.value }),
    });
    const data = (await response.json()) as CreateURLResponse;
    if (!response.ok) {
      throw new Error(data.error?.message || "Short URL gagal dibuat.");
    }
    shortUrl.value = data.short_url;
  } catch (err) {
    error.value = err.message;
  } finally {
    loading.value = false;
  }
}

async function copyUrl() {
  await navigator.clipboard.writeText(shortUrl.value);
  copied.value = true;
}
</script>

<template>
  <main class="min-h-screen bg-stone-100 px-5 py-8 text-slate-900 sm:px-8 sm:py-12">
    <section class="mx-auto w-full max-w-2xl">
      <header class="mb-12 flex items-center justify-between border-b border-stone-300 pb-5">
        <p class="text-lg font-semibold tracking-tight">shortly.</p>
        <p class="text-sm text-slate-500">URL shortener</p>
      </header>

      <div class="mb-8 max-w-xl">
        <p class="mb-3 text-sm font-medium text-slate-500">Mulai di sini</p>
        <h1 class="text-3xl font-semibold tracking-tight sm:text-4xl">Bagikan link tanpa berantakan.</h1>
        <p class="mt-4 max-w-lg leading-7 text-slate-600">Ubah URL panjang menjadi tautan singkat yang mudah dibagikan.</p>
      </div>

      <div class="rounded-xl border border-stone-300 bg-white p-5 shadow-sm sm:p-7">
        <div v-if="error" class="mb-5 rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700" role="alert">
          {{ error }}
        </div>
        <div v-if="shortUrl" class="mb-5 rounded-md border border-green-200 bg-green-50 p-4">
          <p class="text-sm font-medium text-green-800">Short URL berhasil dibuat</p>
          <div class="mt-2 flex flex-col gap-3 sm:flex-row">
            <input v-model="shortUrl" class="min-w-0 flex-1 rounded-md border border-green-200 bg-white px-3 py-2 text-green-900 outline-none" readonly>
            <button type="button" class="rounded-md bg-slate-900 px-4 py-2 font-medium text-white hover:bg-slate-700" @click="copyUrl">
              {{ copied ? "Tersalin" : "Salin" }}
            </button>
          </div>
        </div>
        <form class="space-y-4" @submit.prevent="createShortUrl">
          <label for="original_url" class="block text-sm font-medium text-slate-700">URL tujuan</label>
          <div class="flex flex-col gap-3 sm:flex-row">
            <input id="original_url" v-model="originalUrl" type="url" required maxlength="2048" placeholder="https://contoh.com/halaman-panjang" class="min-w-0 flex-1 rounded-md border border-stone-300 bg-stone-50 px-4 py-3 text-slate-900 outline-none placeholder:text-slate-400 focus:border-slate-500 focus:ring-1 focus:ring-slate-500">
            <button type="submit" :disabled="loading" class="rounded-md bg-slate-900 px-5 py-3 font-medium text-white hover:bg-slate-700 disabled:cursor-not-allowed disabled:opacity-60">
              {{ loading ? "Membuat..." : "Buat URL" }}
            </button>
          </div>
        </form>
      </div>
    </section>
  </main>
</template>
