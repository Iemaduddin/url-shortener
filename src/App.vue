<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import AppSidebar from "./components/AppSidebar.vue";
import AppTopbar from "./components/AppTopbar.vue";
import AuthView from "./components/AuthView.vue";
import AuditLogPage from "./components/AuditLogPage.vue";
import DashboardPage from "./components/DashboardPage.vue";
import UrlListPage from "./components/UrlListPage.vue";
import UrlDetailPage from "./components/UrlDetailPage.vue";
import UserManagementPage from "./components/UserManagementPage.vue";
import type { APIResponse, AuditLog, AuditLogResponse, Page, QRConfig, URLItem, URLListResponse, UserItem, UserListResponse } from "./types";

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
const urlPage = ref(1);
const urlTotalItems = ref(0);
const urlPageSize = 10;
const users = ref<UserItem[]>([]);
const usersLoading = ref(false);
const usersError = ref("");
const userSearch = ref("");
const updatingUserID = ref("");
const userPage = ref(1);
const userTotalItems = ref(0);
const userPageSize = 10;
const detailUrl = ref<URLItem | null>(null);
const detailLoading = ref(false);
const detailError = ref("");
const qrConfig = ref<QRConfig | null>(null);
const qrVersion = ref("");
const qrSaving = ref(false);
const qrMessage = ref("");
const qrError = ref("");
const auditLogs = ref<AuditLog[]>([]);
const auditLoading = ref(false);
const auditError = ref("");
const auditPage = ref(1);
const auditTotalItems = ref(0);
const auditPageSize = 10;
const loginReturnPathKey = "shortly-login-return-path";

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
const urlTotalPages = computed(() => Math.max(1, Math.ceil(urlTotalItems.value / urlPageSize)));
const filteredUsers = computed(() => {
  const query = userSearch.value.trim().toLowerCase();
  return query ? users.value.filter((item) => `${item.email} ${item.role}`.toLowerCase().includes(query)) : users.value;
});
const userTotalPages = computed(() => Math.max(1, Math.ceil(userTotalItems.value / userPageSize)));
const auditTotalPages = computed(() => Math.max(1, Math.ceil(auditTotalItems.value / auditPageSize)));

function getErrorMessage(error: unknown, fallback: string): string {
  return error instanceof Error ? error.message : fallback;
}

function csrfToken(): string {
  const cookie = document.cookie.split(";").map((part) => part.trim()).find((part) => part.startsWith("csrf="));
  return cookie ? decodeURIComponent(cookie.slice("csrf=".length)) : "";
}

function navigate(page: Page) {
  if ((page === "users" || page === "audit") && role.value !== "admin") return;
  currentPage.value = page;
  mobileMenuOpen.value = false;
  const path = page === "dashboard" ? "/dashboard" : `/${page}`;
  if (window.location.pathname !== path) {
    window.history.pushState({ page }, "", path);
  }
  if (page === "users") void loadUsers();
  if (page === "audit") void loadAuditLogs();
}

function pageFromPath(pathname: string): Page {
  if (pathname.startsWith("/urls/")) return "url-detail";
  if (pathname === "/users") return "users";
  if (pathname === "/audit") return "audit";
  if (pathname === "/urls") return "urls";
  return "dashboard";
}

function detailIDFromPath(pathname: string): string {
  return pathname.startsWith("/urls/") ? pathname.slice("/urls/".length) : "";
}

async function loadQRConfig(id: string) {
  try {
    const response = await fetch(`/api/urls/${id}/qr`);
    if (!response.ok) return;
    const data = (await response.json().catch(() => null)) as QRConfig | null;
    if (!data) return;
    qrConfig.value = data;
    qrVersion.value = data.updated_at;
  } catch {
    // Defaults remain available when QR configuration cannot be loaded.
  }
}

async function loadDetail(id: string) {
  detailLoading.value = true;
  detailError.value = "";
  qrConfig.value = null;
  qrVersion.value = Date.now().toString();
  qrMessage.value = "";
  qrError.value = "";
  try {
    const response = await fetch(`/api/urls/${id}`);
    const data = (await response.json().catch(() => null)) as APIResponse | URLItem | null;
    if (!response.ok) {
      const apiError = data && "error" in data ? data.error?.message : undefined;
      throw new Error(apiError || (response.status === 404 ? "URL tidak ditemukan." : "Detail URL gagal dimuat."));
    }
    if (!data || !("id" in data)) throw new Error("Respons detail URL tidak valid.");
    detailUrl.value = data;
    void loadQRConfig(id);
  } catch (error: unknown) {
    detailUrl.value = null;
    detailError.value = getErrorMessage(error, "Detail URL gagal dimuat.");
  } finally {
    detailLoading.value = false;
  }
}

function openDetail(id: string) {
  currentPage.value = "url-detail";
  mobileMenuOpen.value = false;
  window.history.pushState({ page: "url-detail", id }, "", `/urls/${id}`);
  void loadDetail(id);
}

function closeDetail() {
  navigate("urls");
}

async function loadUrls() {
  urlsLoading.value = true;
  try {
    const params = new URLSearchParams({ page: String(urlPage.value), page_size: String(urlPageSize) });
    if (urlSearch.value.trim()) params.set("search", urlSearch.value.trim());
    if (urlStatus.value !== "all") params.set("status", urlStatus.value);
    const response = await fetch(`/api/urls?${params.toString()}`);
    if (response.ok) {
      const data = (await response.json()) as URLListResponse;
      urls.value = data.items;
      urlTotalItems.value = data.total_items;
    }
  } finally {
    urlsLoading.value = false;
  }
}

function updateUrlSearch(value: string) {
  urlSearch.value = value;
  urlPage.value = 1;
  void loadUrls();
}

function updateUrlStatus(value: string) {
  urlStatus.value = value;
  urlPage.value = 1;
  void loadUrls();
}

function updateUrlPage(value: number) {
  urlPage.value = value;
  void loadUrls();
}

async function loadUsers() {
  if (role.value !== "admin") return;
  usersLoading.value = true;
  usersError.value = "";
  try {
    const params = new URLSearchParams({ page: String(userPage.value), page_size: String(userPageSize) });
    if (userSearch.value.trim()) params.set("search", userSearch.value.trim());
    const response = await fetch(`/api/users?${params.toString()}`);
    if (!response.ok) throw new Error("Daftar user gagal dimuat.");
    const data = (await response.json()) as UserListResponse;
    users.value = data.items;
    userTotalItems.value = data.total_items;
  } catch (error: unknown) {
    usersError.value = getErrorMessage(error, "Daftar user gagal dimuat.");
  } finally {
    usersLoading.value = false;
  }
}

function updateUserSearch(value: string) {
  userSearch.value = value;
  userPage.value = 1;
  void loadUsers();
}

function updateUserPage(value: number) {
  userPage.value = value;
  void loadUsers();
}

async function loadAuditLogs() {
  if (role.value !== "admin") return;
  auditLoading.value = true;
  auditError.value = "";
  try {
    const response = await fetch(`/api/audit-logs?page=${auditPage.value}&page_size=${auditPageSize}`);
    const data = (await response.json().catch(() => null)) as AuditLogResponse | APIResponse | null;
    if (!response.ok) {
      const message = data && "error" in data ? data.error?.message : undefined;
      throw new Error(message || `Audit log gagal dimuat (${response.status}).`);
    }
    if (!data || !("items" in data)) throw new Error("Respons audit log tidak valid.");
    auditLogs.value = data.items;
    auditTotalItems.value = data.total_items;
  } catch (error: unknown) {
    auditError.value = getErrorMessage(error, "Audit log gagal dimuat.");
  } finally {
    auditLoading.value = false;
  }
}

function updateAuditPage(value: number) {
  auditPage.value = value;
  void loadAuditLogs();
}

async function configureQR(id: string, form: FormData) {
  qrSaving.value = true;
  qrMessage.value = "";
  qrError.value = "";
  try {
    const response = await fetch(`/api/urls/${id}/qr`, { method: "PUT", headers: { "X-CSRF-Token": csrfToken() }, body: form });
    if (!response.ok) {
      const data = (await response.json().catch(() => null)) as APIResponse | null;
      throw new Error(data?.error?.message || "QR code gagal disimpan.");
    }
    qrVersion.value = Date.now().toString();
    qrMessage.value = "QR code berhasil diperbarui.";
    void loadQRConfig(id);
  } catch (error: unknown) {
    qrError.value = getErrorMessage(error, "QR code gagal disimpan.");
  } finally {
    qrSaving.value = false;
  }
}

async function loadSession() {
  checkingSession.value = true;
  try {
    const response = await fetch("/api/auth/me");
    if (!response.ok) {
      if (window.location.pathname !== "/login") {
        sessionStorage.setItem(loginReturnPathKey, `${window.location.pathname}${window.location.search}${window.location.hash}`);
      }
      if (window.location.pathname !== "/login") {
        window.history.replaceState({ page: "login" }, "", "/login");
      }
      return;
    }
    const data = (await response.json()) as APIResponse;
    authenticated.value = true;
    role.value = data.role || "user";
    const savedPath = sessionStorage.getItem(loginReturnPathKey);
    const requestedPath = window.location.pathname === "/login"
      ? (savedPath?.startsWith("/") && !savedPath.startsWith("//") ? savedPath : "/dashboard")
      : `${window.location.pathname}${window.location.search}${window.location.hash}`;
    sessionStorage.removeItem(loginReturnPathKey);
    const requestedPage = pageFromPath(new URL(requestedPath, window.location.origin).pathname);
    currentPage.value = (requestedPage === "users" || requestedPage === "audit") && role.value !== "admin" ? "dashboard" : requestedPage;
    const targetPath = new URL(requestedPath, window.location.origin).pathname;
    const detailID = detailIDFromPath(targetPath);
    if (currentPage.value === "url-detail" && detailID) void loadDetail(detailID);
    const resolvedPath = currentPage.value === "dashboard" ? "/dashboard" : requestedPath;
    if (window.location.pathname !== resolvedPath) {
      window.history.replaceState({ page: currentPage.value }, "", resolvedPath);
    }
    await loadUrls();
    if (role.value === "admin") await loadUsers();
    if (role.value === "admin" && currentPage.value === "audit") await loadAuditLogs();
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
  await fetch("/api/auth/logout", { method: "POST", headers: { "X-CSRF-Token": csrfToken() } });
  authenticated.value = false;
  role.value = "user";
  urls.value = [];
  users.value = [];
  currentPage.value = "dashboard";
  sessionStorage.removeItem(loginReturnPathKey);
  window.history.replaceState({ page: "login" }, "", "/login");
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
      headers: { "Content-Type": "application/json", "X-CSRF-Token": csrfToken() },
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
    const response = await fetch(`/api/urls/${id}`, { method: "DELETE", headers: { "X-CSRF-Token": csrfToken() } });
    if (response.ok) {
      urls.value = urls.value.map((item) => item.id === id ? { ...item, status: "disabled" } : item);
      if (detailUrl.value?.id === id) detailUrl.value = { ...detailUrl.value, status: "disabled" };
    }
  } finally {
    disablingID.value = "";
  }
}

async function enableUrl(id: string) {
  disablingID.value = id;
  try {
    const response = await fetch(`/api/urls/${id}`, { method: "PUT", headers: { "X-CSRF-Token": csrfToken() } });
    if (response.ok) {
      urls.value = urls.value.map((item) => item.id === id ? { ...item, status: "active" } : item);
      if (detailUrl.value?.id === id) detailUrl.value = { ...detailUrl.value, status: "active" };
    }
  } finally {
    disablingID.value = "";
  }
}

async function createManagedUser(payload: { email: string; password: string; role: string }) {
  updatingUserID.value = "new";
  try {
    const response = await fetch("/api/users", { method: "POST", headers: { "Content-Type": "application/json", "X-CSRF-Token": csrfToken() }, body: JSON.stringify(payload) });
    if (response.ok) await loadUsers();
  } finally {
    updatingUserID.value = "";
  }
}

async function updateManagedUser(user: UserItem, payload: { email: string; password: string; role: string }) {
  updatingUserID.value = user.id;
  try {
    const response = await fetch(`/api/users/${user.id}`, {
      method: "PATCH", headers: { "Content-Type": "application/json", "X-CSRF-Token": csrfToken() }, body: JSON.stringify(payload),
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
      headers: { "Content-Type": "application/json", "X-CSRF-Token": csrfToken() },
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
    const response = await fetch(`/api/users/${user.id}`, { method: "DELETE", headers: { "X-CSRF-Token": csrfToken() } });
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
    if (!authenticated.value && window.location.pathname !== "/login") {
      sessionStorage.setItem(loginReturnPathKey, `${window.location.pathname}${window.location.search}${window.location.hash}`);
      window.history.replaceState({ page: "login" }, "", "/login");
      return;
    }
    const page = pageFromPath(window.location.pathname);
    if (page !== "users" || role.value === "admin") {
      currentPage.value = page;
      if (page === "users") void loadUsers();
      if (page === "audit") void loadAuditLogs();
      if (page === "url-detail") {
        const id = detailIDFromPath(window.location.pathname);
        if (id) void loadDetail(id);
      }
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
        <UrlListPage v-else-if="currentPage === 'urls'" :urls="urls" :filtered-urls="filteredUrls" :urls-loading="urlsLoading" :search="urlSearch" :status="urlStatus" :disabling-id="disablingID" :page="urlPage" :total-pages="urlTotalPages" @update:search="updateUrlSearch" @update:status="updateUrlStatus" @update:page="updateUrlPage" @navigate="navigate" @detail="openDetail" @copy="copyText" @disable="disableUrl" @enable="enableUrl" />
        <UrlDetailPage v-else-if="currentPage === 'url-detail'" :item="detailUrl" :loading="detailLoading" :error="detailError" :qr-config="qrConfig" :qr-version="qrVersion" :qr-saving="qrSaving" :qr-message="qrMessage" :qr-error="qrError" @back="closeDetail" @copy="copyText" @disable="disableUrl" @enable="enableUrl" @configure="configureQR" />
        <UserManagementPage v-else-if="currentPage === 'users'" :users="users" :filtered-users="filteredUsers" :users-loading="usersLoading" :error="usersError" :search="userSearch" :updating-id="updatingUserID" :page="userPage" :total-pages="userTotalPages" @update:search="updateUserSearch" @update:page="updateUserPage" @create="createManagedUser" @update="updateManagedUser" @status="updateUserStatus" @remove="deleteManagedUser" />
        <AuditLogPage v-else :logs="auditLogs" :loading="auditLoading" :error="auditError" :page="auditPage" :total-pages="auditTotalPages" @update:page="updateAuditPage" />
      </div></main>
      <footer class="border-t border-stone-300 bg-white px-5 py-4 text-center text-xs text-slate-500 sm:px-8">shortly. &copy; 2026 · URL management workspace</footer>
    </div>
  </div>
</template>
