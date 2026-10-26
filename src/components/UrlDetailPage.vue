<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from "vue";
import type { QRConfig, URLItem } from "../types";

const props = defineProps<{
  item: URLItem | null;
  loading: boolean;
  error: string;
  qrConfig: QRConfig | null;
  qrVersion: string;
  qrSaving: boolean;
  qrMessage: string;
  qrError: string;
}>();
const emit = defineEmits<{ back: []; copy: [value: string]; disable: [id: string]; enable: [id: string]; configure: [id: string, form: FormData] }>();
const foreground = ref("#000000");
const background = ref("#ffffff");
const frameText = ref("Scan untuk membuka URL");
const logo = ref<File | null>(null);
const logoPreviewURL = ref("");
const previewURL = ref("");
const previewError = ref("");
let previewTimer: ReturnType<typeof setTimeout> | undefined;
let previewRequest = 0;
const qrImageURL = computed(() => {
  if (previewURL.value) return previewURL.value;
  if (!props.item) return "";
  const version = props.qrVersion ? `?v=${encodeURIComponent(props.qrVersion)}` : "";
  return `/qr/${props.item.id}.png${version}`;
});

watch(
  () => props.qrConfig,
  (config) => {
    foreground.value = config?.foreground || "#000000";
    background.value = config?.background || "#ffffff";
    frameText.value = config?.frame_text || "Scan untuk membuka URL";
  },
  { immediate: true },
);

function qrForm(): FormData {
  const form = new FormData();
  form.set("foreground", foreground.value);
  form.set("background", background.value);
  form.set("frame_text", frameText.value);
  if (logo.value) form.set("logo", logo.value);
  return form;
}

function clearPreview() {
  if (previewURL.value) URL.revokeObjectURL(previewURL.value);
  previewURL.value = "";
}

function clearLogoPreview() {
  if (logoPreviewURL.value) URL.revokeObjectURL(logoPreviewURL.value);
  logoPreviewURL.value = "";
}

async function previewErrorMessage(response: Response): Promise<string> {
  if (response.status === 429) return "Terlalu banyak permintaan untuk pratinjau QR. Coba lagi dalam satu menit.";
  const data = (await response.json().catch(() => null)) as { error?: { message?: string } } | null;
  return data?.error?.message || "Pratinjau QR gagal diperbarui. Coba lagi.";
}

async function refreshPreview() {
  if (!props.item) return;
  const request = ++previewRequest;
  try {
    const response = await fetch(`/api/urls/${props.item.id}/qr/preview`, { method: "PUT", body: qrForm() });
    if (!response.ok) {
      if (request === previewRequest) previewError.value = await previewErrorMessage(response);
      return;
    }
    const blob = await response.blob();
    if (request !== previewRequest || !blob.type.startsWith("image/")) return;
    const previous = previewURL.value;
    previewURL.value = URL.createObjectURL(blob);
    previewError.value = "";
    if (previous) {
      await nextTick();
      URL.revokeObjectURL(previous);
    }
  } catch {
    // Keep the last successful preview when a new one cannot be generated.
    if (request === previewRequest) previewError.value = "Pratinjau QR gagal diperbarui. Periksa koneksi Anda.";
  }
}

watch([foreground, background, frameText, logo], () => {
  clearTimeout(previewTimer);
  previewTimer = setTimeout(() => void refreshPreview(), 300);
});

watch(
  () => props.item?.id,
  () => {
    previewRequest++;
    previewError.value = "";
    clearPreview();
    clearLogoPreview();
    logo.value = null;
  },
);

onBeforeUnmount(() => {
  previewRequest++;
  clearTimeout(previewTimer);
  clearPreview();
  clearLogoPreview();
});

function share(item: URLItem) {
  if (navigator.share) {
    void navigator.share({ title: item.short_url, url: item.short_url });
    return;
  }
  emit("copy", item.short_url);
}

function selectLogo(event: Event) {
  const file = (event.target as HTMLInputElement).files?.[0] || null;
  logo.value = file;
  clearLogoPreview();
  if (file) logoPreviewURL.value = URL.createObjectURL(file);
}

function saveQR() {
  if (!props.item || props.qrSaving) return;
  emit("configure", props.item.id, qrForm());
}
</script>

<template>
  <section>
    <div class="mb-8 flex flex-col justify-between gap-4 sm:flex-row sm:items-end">
      <div>
        <button type="button" class="mb-4 text-sm text-slate-500 hover:text-slate-900" @click="emit('back')">← Kembali ke URL List</button>
        <p class="mb-2 text-sm font-medium text-slate-500">URL Detail</p>
        <h1 class="text-3xl font-semibold tracking-tight sm:text-4xl">Detail URL.</h1>
        <p class="mt-3 text-slate-600">Informasi dan statistik penggunaan tautan.</p>
      </div>
    </div>
    <div v-if="loading" class="rounded-xl border border-stone-300 bg-white p-10 text-center text-sm text-slate-500">Memuat detail URL...</div>
    <div v-else-if="error" class="rounded-xl border border-red-200 bg-red-50 p-5 text-sm text-red-700" role="alert">{{ error }}</div>
    <div v-else-if="item" class="space-y-6">
      <section class="rounded-xl border border-stone-300 bg-white p-5 shadow-sm sm:p-7">
        <div class="flex flex-col justify-between gap-5 sm:flex-row sm:items-start">
          <div class="min-w-0">
            <p class="mb-2 text-xs font-semibold uppercase tracking-widest text-slate-400">Short URL</p>
            <div class="flex items-center gap-3"><a :href="item.short_url" target="_blank" class="truncate text-xl font-semibold text-slate-900 underline underline-offset-4 hover:text-slate-600">{{ item.short_url }}</a><button type="button" class="shrink-0 rounded-md border border-stone-300 px-3 py-1.5 text-xs font-medium hover:bg-stone-50" @click="emit('copy', item.short_url)">Salin</button></div>
            <p class="mt-4 text-xs font-semibold uppercase tracking-widest text-slate-400">URL tujuan</p>
            <p class="mt-2 break-all text-sm leading-6 text-slate-600">{{ item.original_url }}</p>
          </div>
          <span class="self-start rounded-full px-3 py-1.5 text-xs font-semibold uppercase tracking-wider" :class="item.status === 'active' ? 'bg-green-50 text-green-700' : 'bg-stone-100 text-slate-400'">{{ item.status }}</span>
        </div>
        <div class="mt-6 flex flex-wrap gap-3 border-t border-stone-200 pt-5 text-sm">
          <a :href="qrImageURL" download class="rounded-md bg-slate-900 px-3 py-2 text-sm font-medium text-white hover:bg-slate-700">Download QR</a>
          <button type="button" class="rounded-md border border-stone-300 px-3 py-2 text-sm hover:bg-stone-50" @click="share(item)">Bagikan</button>
          <button v-if="item.status === 'active'" type="button" class="text-slate-500 hover:text-red-700" @click="emit('disable', item.id)">Nonaktifkan URL</button>
          <button v-else type="button" class="text-slate-500 hover:text-green-700" @click="emit('enable', item.id)">Aktifkan kembali</button>
        </div>
      </section>
      <section class="flex justify-center rounded-xl border border-stone-300 bg-white p-7 shadow-sm">
        <div class="text-center"><img :src="qrImageURL" alt="QR code short URL" class="mx-auto h-auto w-64 max-w-full rounded-md border border-stone-200 p-2"><p class="mt-4 text-sm text-slate-500">Scan untuk membuka URL</p><p v-if="previewError" class="mt-3 text-sm text-red-700" role="alert">{{ previewError }}</p></div>
      </section>
      <section class="rounded-xl border border-stone-300 bg-white p-5 shadow-sm sm:p-7"><h2 class="font-semibold">Kustomisasi QR</h2><p class="mt-1 text-sm text-slate-500">Perubahan warna, teks, dan logo langsung tampil di pratinjau. Klik Simpan untuk menerapkannya.</p><form class="mt-5 grid gap-4 sm:grid-cols-2" @submit.prevent="saveQR"><label class="text-sm">Warna QR<input v-model="foreground" type="color" class="mt-2 block h-10 w-full rounded border border-stone-300"></label><label class="text-sm">Warna latar<input v-model="background" type="color" class="mt-2 block h-10 w-full rounded border border-stone-300"></label><label class="text-sm sm:col-span-2">Teks frame<input v-model="frameText" maxlength="100" class="mt-2 w-full rounded-md border border-stone-300 px-3 py-2"></label><label class="text-sm sm:col-span-2">Logo PNG/JPEG/WebP, maksimal 5 MB<img v-if="logoPreviewURL" :src="logoPreviewURL" alt="Preview logo terpilih" class="mt-2 h-24 w-24 rounded-md border border-stone-200 bg-stone-50 object-contain p-1"><input type="file" accept="image/png,image/jpeg,image/webp" class="mt-2 block w-full cursor-pointer rounded-md border border-stone-300 bg-white px-3 py-2 text-sm text-slate-600 file:mr-3 file:cursor-pointer file:rounded-md file:border file:border-stone-300 file:bg-stone-50 file:px-3 file:py-1.5 file:text-xs file:font-semibold file:text-slate-700 hover:file:bg-stone-100 hover:file:border-slate-400" @change="selectLogo"></label><button type="submit" :disabled="qrSaving" class="rounded-md bg-slate-900 px-4 py-2 text-sm font-medium text-white disabled:opacity-60 sm:col-span-2 sm:w-fit">{{ qrSaving ? "Menyimpan..." : "Simpan QR custom" }}</button></form><p v-if="qrMessage" class="mt-3 text-sm text-green-700" role="status">{{ qrMessage }}</p><p v-if="qrError" class="mt-3 text-sm text-red-700" role="alert">{{ qrError }}</p><p v-if="qrConfig?.has_logo" class="mt-3 text-sm text-slate-500">Logo tersimpan. Unggah logo baru untuk menggantinya.</p></section>
      <section class="grid gap-4 sm:grid-cols-3">
        <div class="rounded-xl border border-stone-300 bg-white p-5"><p class="text-sm text-slate-500">Total klik</p><p class="mt-3 text-3xl font-semibold">{{ item.total_clicks }}</p><p class="mt-2 text-xs text-slate-400">Semua redirect yang tercatat</p></div>
        <div class="rounded-xl border border-stone-300 bg-white p-5"><p class="text-sm text-slate-500">Klik terakhir</p><p class="mt-3 text-lg font-semibold">{{ item.last_clicked_at ? new Date(item.last_clicked_at).toLocaleString("id-ID") : "Belum ada" }}</p><p class="mt-2 text-xs text-slate-400">Waktu akses terbaru</p></div>
        <div class="rounded-xl border border-stone-300 bg-white p-5"><p class="text-sm text-slate-500">Dibuat</p><p class="mt-3 text-lg font-semibold">{{ new Date(item.created_at).toLocaleDateString("id-ID", { day: "numeric", month: "short", year: "numeric" }) }}</p><p class="mt-2 text-xs text-slate-400">Tanggal pembuatan URL</p></div>
      </section>
    </div>
  </section>
</template>
