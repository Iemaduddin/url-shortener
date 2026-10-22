<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import AppSidebar from "./components/AppSidebar.vue";
import AppTopbar from "./components/AppTopbar.vue";
import AuthView from "./components/AuthView.vue";
import DashboardPage from "./components/DashboardPage.vue";
import UrlListPage from "./components/UrlListPage.vue";
import UserManagementPage from "./components/UserManagementPage.vue";
import type { APIResponse, Page, URLItem, UserItem } from "./types";

const authenticated = ref(false);
const checkingSession = ref(true);
const authMode = ref<"login" | "register">("login");
const email = ref("");
const password = ref("");
const role = ref("user");
const authError = ref("");
const authMessage = ref("");
const authLoading = ref(false);
const currentPage = ref<Page>("dashboard");
const mobileMenuOpen = ref(false);
const originalUrl = ref("");
const shortUrl = ref("");
const copied = ref(false);
const urlError = ref("");
const urlMessage = ref("");
const urlLoading = ref(false);
const urls = ref<URLItem[]>([]);
const urlsLoading = ref(false);
const disablingID = ref("");
const urlSearch = ref("");
const urlStatus = ref("all");
const users = ref<UserItem[]>([]);
const usersLoading = ref(false);
const usersError = ref("");
const userSearch = ref("");
const updatingUserID = ref("");

const activeUrls = computed(() => urls.value.filter((item) => item.status === "active"));
const disabledUrls = computed(() => urls.value.filter((item) => item.status !== "active"));
const filteredUrls = computed(() => {
  const query = urlSearch.value.trim().toLowerCase();
  return urls.value.filter((item) => {
    const matchesStatus = urlStatus.value === "all" || item.status === urlStatus.value;
    const matchesQuery = !query || `${item.short_url} ${item.original_url}`.toLowerCase().includes(query);
    return matchesStatus && matchesQuery;
  });
});
const filteredUsers = computed(() => {
  const query = userSearch.value.trim().toLowerCase();
  return query ? users.value.filter((item) => `${item.email} ${item.role}`.toLowerCase().includes(query)) : users.value;
});

function getErrorMessage(error: unknown, fallback: string): string {
  return error instanceof Error ? error.message : fallback;
}

function navigate(page: Page) {
  if (page === "users" && role.value !== "admin") return;
  currentPage.value = page;
  mobileMenuOpen.value = false;
  const path = page === "dashboard" ? "/dashboard" : `/${page}`;
  if (window.location.pathname !== path) {
    window.history.pushState({ page }, "", path);
  }
  if (page === "users") void loadUsers();
}

function pageFromPath(pathname: string): Page {
  if (pathname === "/users") return "users";
  if (pathname === "/urls") return "urls";
  return "dashboard";
}

async function loadUrls() {
  urlsLoading.value = true;
  try {
    const response = await fetch("/api/urls");
    if (response.ok) urls.value = (await response.json()) as URLItem[];
  } finally {
    urlsLoading.value = false;
  }
}

async function loadUsers() {
  if (role.value !== "admin") return;
  usersLoading.value = true;
  usersError.value = "";
  try {
    const response = await fetch("/api/users");
    if (!response.ok) throw new Error("Daftar user gagal dimuat.");
    users.value = (await response.json()) as UserItem[];
  } catch (error: unknown) {
    usersError.value = getErrorMessage(error, "Daftar user gagal dimuat.");
  } finally {
    usersLoading.value = false;
  }
}

async function loadSession() {
  checkingSession.value = true;
  try {
    const response = await fetch("/api/auth/me");
    if (!response.ok) return;
    const data = (await response.json()) as APIResponse;
    authenticated.value = true;
    role.value = data.role || "user";
    const requestedPage = pageFromPath(window.location.pathname);
    currentPage.value = requestedPage === "users" && role.value !== "admin" ? "dashboard" : requestedPage;
    const resolvedPath = currentPage.value === "dashboard" ? "/dashboard" : `/${currentPage.value}`;
    if (window.location.pathname !== resolvedPath) {
      window.history.replaceState({ page: currentPage.value }, "", resolvedPath);
    }
    await loadUrls();
    if (role.value === "admin") await loadUsers();
  } finally {
    checkingSession.value = false;
  }
}

function switchAuthMode(mode: "login" | "register") {
  authMode.value = mode;
  authError.value = "";
  authMessage.value = "";
}

async function submitAuth() {
  authError.value = "";
  authMessage.value = "";
  authLoading.value = true;
  try {
    const response = await fetch(`/api/auth/${authMode.value}`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ email: email.value, password: password.value }),
    });
    const data = (await response.json().catch(() => ({}))) as APIResponse;
    if (!response.ok) throw new Error(data.error?.message || "Autentikasi gagal.");
    if (authMode.value === "register") {
      authMode.value = "login";
      password.value = "";
      authMessage.value = "Akun berhasil dibuat. Silakan masuk.";
      return;
    }
    password.value = "";
    await loadSession();
  } catch (error: unknown) {
    authError.value = getErrorMessage(error, "Autentikasi gagal.");
  } finally {
    authLoading.value = false;
  }
}

async function logout() {
  await fetch("/api/auth/logout", { method: "POST" });
  authenticated.value = false;
  role.value = "user";
  urls.value = [];
  users.value = [];
  currentPage.value = "dashboard";
  window.history.replaceState({ page: "dashboard" }, "", "/dashboard");
}

async function createShortUrl() {
  urlError.value = "";
  urlMessage.value = "";
  shortUrl.value = "";
  copied.value = false;
  urlLoading.value = true;
  try {
    const response = await fetch("/api/urls", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ original_url: originalUrl.value }),
    });
    const data = (await response.json().catch(() => ({}))) as APIResponse;
    if (!response.ok) throw new Error(data.error?.message || "Short URL gagal dibuat.");
    shortUrl.value = data.short_url || "";
    urlMessage.value = "URL berhasil dibuat dan siap dibagikan.";
    originalUrl.value = "";
    await loadUrls();
  } catch (error: unknown) {
    urlError.value = getErrorMessage(error, "Short URL gagal dibuat.");
  } finally {
    urlLoading.value = false;
  }
}

async function disableUrl(id: string) {
  disablingID.value = id;
  try {
    const response = await fetch(`/api/urls/${id}`, { method: "DELETE" });
    if (response.ok) urls.value = urls.value.map((item) => item.id === id ? { ...item, status: "disabled" } : item);
  } finally {
    disablingID.value = "";
  }
}

async function enableUrl(id: string) {
  disablingID.value = id;
  try {
    const response = await fetch(`/api/urls/${id}`, { method: "PUT" });
    if (response.ok) urls.value = urls.value.map((item) => item.id === id ? { ...item, status: "active" } : item);
  } finally {
    disablingID.value = "";
  }
}

async function createManagedUser(payload: { email: string; password: string; role: string }) {
  updatingUserID.value = "new";
  try {
    const response = await fetch("/api/users", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(payload) });
    if (response.ok) await loadUsers();
  } finally {
    updatingUserID.value = "";
  }
}

async function updateManagedUser(user: UserItem, payload: { email: string; password: string; role: string }) {
  updatingUserID.value = user.id;
  try {
    const response = await fetch(`/api/users/${user.id}`, {
      method: "PATCH", headers: { "Content-Type": "application/json" }, body: JSON.stringify(payload),
    });
    if (response.ok) await loadUsers();
  } finally {
    updatingUserID.value = "";
  }
}

async function updateUserStatus(user: UserItem, status: string) {
  updatingUserID.value = user.id;
  try {
    const response = await fetch(`/api/users/${user.id}`, {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ status }),
    });
    if (response.ok) users.value = users.value.map((item) => item.id === user.id ? { ...item, status } : item);
  } finally {
    updatingUserID.value = "";
  }
}

async function deleteManagedUser(user: UserItem) {
  if (!window.confirm(`Hapus ${user.email} secara permanen?`)) return;
  updatingUserID.value = user.id;
  try {
    const response = await fetch(`/api/users/${user.id}`, { method: "DELETE" });
    if (response.ok) users.value = users.value.filter((item) => item.id !== user.id);
  } finally {
    updatingUserID.value = "";
  }
}

async function copyText(value: string) {
  await navigator.clipboard.writeText(value);
  if (value === shortUrl.value) copied.value = true;
}

onMounted(() => {
  window.addEventListener("popstate", () => {
    const page = pageFromPath(window.location.pathname);
    if (page !== "users" || role.value === "admin") {
      currentPage.value = page;
      if (page === "users") void loadUsers();
    }
  });
  void loadSession();
});
</script>

<template>
  <div v-if="checkingSession" class="flex min-h-screen items-center justify-center bg-[#f7f7f5] text-sm text-slate-500">Memeriksa sesi...</div>
  <AuthView v-else-if="!authenticated" :mode="authMode" :email="email" :password="password" :error="authError" :message="authMessage" :loading="authLoading" @update:mode="switchAuthMode" @update:email="email = $event" @update:password="password = $event" @submit="submitAuth" />
  <div v-else class="flex min-h-screen bg-[#f7f7f5] text-slate-900">
    <AppSidebar :role="role" :current-page="currentPage" @navigate="navigate" />
    <div class="flex min-w-0 flex-1 flex-col">
      <AppTopbar :page="currentPage" :role="role" :mobile-menu-open="mobileMenuOpen" @navigate="navigate" @toggle-menu="mobileMenuOpen = !mobileMenuOpen" @logout="logout" />
      <main class="flex-1 px-5 py-8 sm:px-8"><div class="mx-auto">
        <DashboardPage v-if="currentPage === 'dashboard'" :urls="urls" :active-urls="activeUrls" :disabled-urls="disabledUrls" :urls-loading="urlsLoading" :original-url="originalUrl" :short-url="shortUrl" :copied="copied" :url-error="urlError" :url-message="urlMessage" :url-loading="urlLoading" @update:original-url="originalUrl = $event" @create="createShortUrl" @copy="copyText" @navigate="navigate" />
        <UrlListPage v-else-if="currentPage === 'urls'" :urls="urls" :filtered-urls="filteredUrls" :urls-loading="urlsLoading" :search="urlSearch" :status="urlStatus" :disabling-id="disablingID" @update:search="urlSearch = $event" @update:status="urlStatus = $event" @navigate="navigate" @copy="copyText" @disable="disableUrl" @enable="enableUrl" />
        <UserManagementPage v-else :users="users" :filtered-users="filteredUsers" :users-loading="usersLoading" :error="usersError" :search="userSearch" :updating-id="updatingUserID" @update:search="userSearch = $event" @create="createManagedUser" @update="updateManagedUser" @status="updateUserStatus" @remove="deleteManagedUser" />
      </div></main>
      <footer class="border-t border-stone-300 bg-white px-5 py-4 text-center text-xs text-slate-500 sm:px-8">shortly. &copy; 2026 · URL management workspace</footer>
    </div>
  </div>
</template>
