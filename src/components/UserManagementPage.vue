<script setup lang="ts">
import { ref } from "vue";
import type { UserItem } from "../types";

defineProps<{ users: UserItem[]; filteredUsers: UserItem[]; usersLoading: boolean; error: string; search: string; updatingId: string }>();
const emit = defineEmits<{ "update:search": [value: string]; create: [payload: { email: string; password: string; role: string }]; update: [user: UserItem, payload: { email: string; password: string; role: string }]; status: [user: UserItem, value: string]; remove: [user: UserItem] }>();
const formOpen = ref(false);
const editing = ref<UserItem | null>(null);
const formEmail = ref("");
const formPassword = ref("");
const formRole = ref("user");

function openCreate() {
  editing.value = null;
  formEmail.value = "";
  formPassword.value = "";
  formRole.value = "user";
  formOpen.value = true;
}

function openEdit(user: UserItem) {
  editing.value = user;
  formEmail.value = user.email;
  formPassword.value = "";
  formRole.value = user.role;
  formOpen.value = true;
}

function submitForm() {
  const payload = { email: formEmail.value, password: formPassword.value, role: formRole.value };
  if (editing.value) emit("update", editing.value, payload);
  else emit("create", payload);
  formOpen.value = false;
}
</script>

<template>
  <section>
    <div class="mb-8 flex flex-col justify-between gap-4 sm:flex-row sm:items-end"><div><p class="mb-2 text-sm font-medium text-slate-500">Administration</p><h1 class="text-3xl font-semibold tracking-tight sm:text-4xl">User Management.</h1><p class="mt-3 text-slate-600">Buat, ubah, nonaktifkan, dan hapus akun dari satu tempat.</p></div><button type="button" class="rounded-md bg-slate-900 px-4 py-2.5 text-sm font-medium text-white hover:bg-slate-700" @click="openCreate">Tambah user</button></div>
    <div class="mb-5 grid gap-4 sm:grid-cols-3"><div class="rounded-lg border border-stone-300 bg-white px-4 py-3"><p class="text-xs text-slate-500">Total user</p><p class="mt-1 text-xl font-semibold">{{ users.length }}</p></div><div class="rounded-lg border border-stone-300 bg-white px-4 py-3"><p class="text-xs text-slate-500">Admin</p><p class="mt-1 text-xl font-semibold">{{ users.filter((item) => item.role === 'admin').length }}</p></div><div class="rounded-lg border border-stone-300 bg-white px-4 py-3"><p class="text-xs text-slate-500">User biasa</p><p class="mt-1 text-xl font-semibold">{{ users.filter((item) => item.role === 'user').length }}</p></div></div>
    <section class="overflow-hidden rounded-xl border border-stone-300 bg-white shadow-sm"><div class="flex flex-col gap-3 border-b border-stone-200 p-4 sm:flex-row sm:items-center sm:justify-between"><div><h2 class="font-semibold">Daftar akun</h2><p class="mt-1 text-xs text-slate-500">{{ filteredUsers.length }} akun ditemukan</p></div><input :value="search" type="search" placeholder="Cari email atau role..." class="w-full rounded-md border border-stone-300 bg-stone-50 px-3 py-2 text-sm outline-none focus:border-slate-500 sm:w-64" @input="emit('update:search', ($event.target as HTMLInputElement).value)"></div><div v-if="error" class="m-4 rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700" role="alert">{{ error }}</div><div v-else-if="usersLoading" class="p-8 text-center text-sm text-slate-500">Memuat user...</div><div v-else-if="filteredUsers.length === 0" class="p-10 text-center text-sm text-slate-500">Belum ada user yang cocok.</div><div v-else class="overflow-x-auto"><table class="w-full min-w-[900px] text-left text-sm"><thead class="border-b border-stone-200 bg-stone-50 text-xs uppercase tracking-wide text-slate-500"><tr><th class="px-5 py-3 font-medium">User</th><th class="px-5 py-3 font-medium">Role</th><th class="px-5 py-3 font-medium">Status</th><th class="px-5 py-3 font-medium">Bergabung</th><th class="px-5 py-3 text-right font-medium">Aksi</th></tr></thead><tbody class="divide-y divide-stone-200"><tr v-for="item in filteredUsers" :key="item.id" class="hover:bg-stone-50"><td class="px-5 py-4 font-medium text-slate-900">{{ item.email }}</td><td class="px-5 py-4 text-slate-600">{{ item.role }}</td><td class="px-5 py-4"><span class="rounded-full px-2.5 py-1 text-[10px] font-semibold uppercase tracking-wider" :class="item.status === 'active' ? 'bg-green-50 text-green-700' : 'bg-stone-100 text-slate-400'">{{ item.status }}</span></td><td class="px-5 py-4 text-slate-500">{{ new Date(item.created_at).toLocaleDateString("id-ID", { day: "numeric", month: "short", year: "numeric" }) }}</td><td class="space-x-3 px-5 py-4 text-right text-xs"><button type="button" :disabled="updatingId === item.id" class="text-slate-600 hover:text-slate-950 disabled:opacity-50" @click="openEdit(item)">Edit</button><button type="button" :disabled="updatingId === item.id" class="text-slate-600 hover:text-slate-950 disabled:opacity-50" @click="emit('status', item, item.status === 'active' ? 'disabled' : 'active')">{{ item.status === 'active' ? 'Nonaktifkan' : 'Aktifkan' }}</button><button type="button" :disabled="updatingId === item.id" class="text-red-600 hover:text-red-800 disabled:opacity-50" @click="emit('remove', item)">Hapus</button></td></tr></tbody></table></div></section>
    <div v-if="formOpen" class="fixed inset-0 z-20 flex items-center justify-center bg-slate-900/30 px-5"><form class="w-full max-w-md rounded-xl border border-stone-300 bg-white p-6 shadow-xl" @submit.prevent="submitForm"><div class="mb-5 flex items-start justify-between"><div><h2 class="text-xl font-semibold">{{ editing ? "Edit user" : "Tambah user" }}</h2><p class="mt-1 text-sm text-slate-500">{{ editing ? "Kosongkan password jika tidak ingin mengubahnya." : "Password minimal 8 karakter." }}</p></div><button type="button" class="text-slate-400 hover:text-slate-900" @click="formOpen = false">Tutup</button></div><div class="space-y-4"><div><label class="mb-1.5 block text-sm font-medium">Email</label><input v-model="formEmail" type="email" required class="w-full rounded-md border border-stone-300 bg-stone-50 px-3 py-2.5 outline-none focus:border-slate-500"></div><div><label class="mb-1.5 block text-sm font-medium">Password</label><input v-model="formPassword" type="password" :required="!editing" minlength="8" class="w-full rounded-md border border-stone-300 bg-stone-50 px-3 py-2.5 outline-none focus:border-slate-500"></div><div><label class="mb-1.5 block text-sm font-medium">Role</label><select v-model="formRole" class="w-full rounded-md border border-stone-300 bg-stone-50 px-3 py-2.5 outline-none focus:border-slate-500"><option value="user">User</option><option value="admin">Admin</option></select></div></div><div class="mt-6 flex justify-end gap-2"><button type="button" class="rounded-md border border-stone-300 px-4 py-2 text-sm" @click="formOpen = false">Batal</button><button type="submit" class="rounded-md bg-slate-900 px-4 py-2 text-sm font-medium text-white">{{ editing ? "Simpan perubahan" : "Buat user" }}</button></div></form></div>
  </section>
</template>
