<script setup lang="ts">
defineProps<{
  mode: "login" | "register";
  email: string;
  password: string;
  error: string;
  message: string;
  loading: boolean;
}>();

const emit = defineEmits<{
  "update:mode": [value: "login" | "register"];
  "update:email": [value: string];
  "update:password": [value: string];
  submit: [];
}>();
</script>

<template>
  <main class="min-h-screen bg-[#f7f7f5] px-5 py-8 text-slate-900 sm:px-8 sm:py-12">
    <section class="mx-auto w-full max-w-5xl">
      <header class="mb-16 flex items-center justify-between border-b border-stone-300 pb-5">
        <p class="text-lg font-semibold tracking-tight">shortly.</p>
        <span class="text-sm text-slate-500">URL workspace</span>
      </header>
      <div class="mx-auto grid max-w-4xl gap-12 lg:grid-cols-[1fr_360px] lg:items-center">
        <div>
          <p class="mb-4 text-sm font-medium text-slate-500">Link sharing, dibuat sederhana</p>
          <h1 class="max-w-xl text-4xl font-semibold tracking-tight sm:text-5xl">Tautan singkat untuk dibagikan dengan percaya diri.</h1>
          <p class="mt-5 max-w-lg leading-7 text-slate-600">Buat, pantau, dan kelola semua tautanmu dalam satu workspace yang tenang.</p>
        </div>
        <div class="rounded-xl border border-stone-300 bg-white p-6 shadow-sm">
          <div class="mb-6 border-b border-stone-200 pb-4">
            <div class="flex gap-5 text-sm">
              <button type="button" :class="mode === 'login' ? 'border-b-2 border-slate-900 pb-3 font-semibold text-slate-900' : 'pb-3 text-slate-500'" @click="emit('update:mode', 'login')">Masuk</button>
              <button type="button" :class="mode === 'register' ? 'border-b-2 border-slate-900 pb-3 font-semibold text-slate-900' : 'pb-3 text-slate-500'" @click="emit('update:mode', 'register')">Daftar</button>
            </div>
          </div>
          <div v-if="error" class="mb-4 rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700" role="alert">{{ error }}</div>
          <div v-if="message" class="mb-4 rounded-md border border-green-200 bg-green-50 px-3 py-2 text-sm text-green-700" role="status">{{ message }}</div>
          <form class="space-y-4" @submit.prevent="emit('submit')">
            <div><label for="email" class="mb-1.5 block text-sm font-medium text-slate-700">Email</label><input id="email" :value="email" type="email" autocomplete="email" required placeholder="nama@email.com" class="w-full rounded-md border border-stone-300 bg-stone-50 px-3 py-2.5 outline-none focus:border-slate-500 focus:ring-1 focus:ring-slate-500" @input="emit('update:email', ($event.target as HTMLInputElement).value)"></div>
            <div><label for="password" class="mb-1.5 block text-sm font-medium text-slate-700">Password</label><input id="password" :value="password" type="password" minlength="8" required placeholder="Minimal 8 karakter" class="w-full rounded-md border border-stone-300 bg-stone-50 px-3 py-2.5 outline-none focus:border-slate-500 focus:ring-1 focus:ring-slate-500" @input="emit('update:password', ($event.target as HTMLInputElement).value)"></div>
            <button type="submit" :disabled="loading" class="w-full rounded-md bg-slate-900 px-4 py-2.5 font-medium text-white hover:bg-slate-700 disabled:opacity-60">{{ loading ? "Memproses..." : mode === "login" ? "Masuk ke akun" : "Buat akun" }}</button>
          </form>
        </div>
      </div>
    </section>
  </main>
</template>
