<script setup lang="ts">
import { ref, computed } from "vue";
import { useRouter } from "vue-router";
import {
  NLayout, NLayoutHeader, NLayoutContent, NLayoutSider,
  NSpace, NButton, NText, NDataTable, NSelect, NInput, NDatePicker, NForm, NFormItem,
  NTag, NEmpty, NSpin, useMessage,
} from "naive-ui";
import type { DataTableColumns } from "naive-ui";
import { useAuthStore } from "@/stores/auth";
import { useThemeStore } from "@/stores/theme";
import { useEduPanelStore } from "@/stores/eduPanel";
import { suratApi, type SuratListItem, type ListSuratParams } from "@/api/surat";
import { db } from "@/offline/db";
import NotificationBell from "@/components/NotificationBell.vue";
import PendingSyncIndicator from "@/components/PendingSyncIndicator.vue";

const router = useRouter();
const auth = useAuthStore();
const themeStore = useThemeStore();
const eduPanel = useEduPanelStore();
const message = useMessage();

// State
const items = ref<SuratListItem[]>([]);
const loading = ref(false);
const nextCursor = ref<{ created_at: string; id: string } | null>(null);
const hasMore = computed(() => nextCursor.value !== null);

// Filter
const jenisOptions = [
  { label: "Semua", value: "" },
  { label: "Surat Masuk", value: "masuk" },
  { label: "Surat Keluar", value: "keluar" },
];
const filterJenis = ref<"" | "masuk" | "keluar">("");
const filterSearch = ref("");
const filterTanggalRange = ref<[number, number] | null>(null);

const sifatTagType: Record<string, "default" | "info" | "warning" | "error"> = {
  biasa: "default",
  segera: "warning",
  penting: "info",
  rahasia: "error",
};

async function fetchPage(append = false) {
  loading.value = true;
  try {
    const params: ListSuratParams = {
      limit: 20,
    };
    if (filterJenis.value) params.jenis = filterJenis.value;
    if (filterSearch.value.trim()) params.search = filterSearch.value.trim();
    if (filterTanggalRange.value) {
      const [from, to] = filterTanggalRange.value;
      params.tanggal_dari = new Date(from).toISOString().slice(0, 10);
      params.tanggal_sampai = new Date(to).toISOString().slice(0, 10);
    }
    if (append && nextCursor.value) {
      params.after_id = nextCursor.value.id;
      params.after_created_at = nextCursor.value.created_at;
    }
    const resp = await suratApi.list(params);
    items.value = append ? [...items.value, ...resp.items] : resp.items;
    nextCursor.value = resp.next_cursor ?? null;
  } catch (e) {
    // Online API gagal — coba fallback ke IndexedDB cache.
    // Filter & sort di-apply manual (bukan keyset pagination — dataset cache
    // ukurannya kecil, full table scan acceptable).
    if (await tryOfflineFallback(append)) {
      message.warning("Memuat dari cache lokal — koneksi terputus");
    } else {
      message.error("Gagal memuat daftar surat");
      console.error(e);
    }
  } finally {
    loading.value = false;
  }
}

async function tryOfflineFallback(append: boolean): Promise<boolean> {
  try {
    let collection = db.surat.orderBy("tanggal_terima").reverse();
    if (filterJenis.value) {
      const j = filterJenis.value;
      collection = collection.filter((s) => s.jenis === j);
    }
    if (filterSearch.value.trim()) {
      const q = filterSearch.value.trim().toLowerCase();
      collection = collection.filter((s) => s.perihal.toLowerCase().includes(q));
    }
    if (filterTanggalRange.value) {
      const [from, to] = filterTanggalRange.value;
      const fromStr = new Date(from).toISOString().slice(0, 10);
      const toStr = new Date(to).toISOString().slice(0, 10);
      collection = collection.filter(
        (s) => s.tanggal_surat >= fromStr && s.tanggal_surat <= toStr,
      );
    }
    const cached = await collection.limit(20).toArray();
    // Adapt CachedSurat ke SuratListItem (compatible by structure).
    const adapted: SuratListItem[] = cached.map((c) => ({
      id: c.id,
      jenis: c.jenis,
      nomor_surat: c.nomor_surat,
      perihal: c.perihal,
      tanggal_surat: c.tanggal_surat,
      tanggal_terima: c.tanggal_terima,
      instansi_id: c.instansi_id,
      instansi_nama: c.instansi_nama,
      klasifikasi_kode: c.klasifikasi_kode,
      sifat_kode: c.sifat_kode,
      access_level: c.access_level,
      created_at: c.tanggal_surat, // approximation — full created_at tidak di-cache
    }));
    items.value = append ? [...items.value, ...adapted] : adapted;
    nextCursor.value = null; // disable load-more saat offline (no cursor)
    return adapted.length > 0;
  } catch (e) {
    console.error("offline fallback failed:", e);
    return false;
  }
}

function applyFilter() {
  nextCursor.value = null;
  items.value = [];
  fetchPage(false);
}

function loadMore() {
  if (hasMore.value && !loading.value) {
    fetchPage(true);
  }
}

function logout() {
  auth.logout();
  router.push({ name: "login" });
}

// Initial load
fetchPage(false);

const columns: DataTableColumns<SuratListItem> = [
  {
    title: "Jenis",
    key: "jenis",
    width: 110,
    render: (row) => h(NTag, { type: row.jenis === "masuk" ? "success" : "info", size: "small" }, () =>
      row.jenis === "masuk" ? "Masuk" : "Keluar"),
  },
  { title: "Nomor", key: "nomor_surat", width: 200, ellipsis: { tooltip: true } },
  { title: "Perihal", key: "perihal", ellipsis: { tooltip: true } },
  {
    title: "Tanggal",
    key: "tanggal_surat",
    width: 110,
    render: (row) => row.tanggal_surat,
  },
  {
    title: "Instansi",
    key: "instansi_nama",
    width: 200,
    ellipsis: { tooltip: true },
  },
  {
    title: "Sifat",
    key: "sifat_kode",
    width: 100,
    render: (row) =>
      row.sifat_kode
        ? h(NTag, { type: sifatTagType[row.sifat_kode] ?? "default", size: "small" }, () => row.sifat_kode!)
        : "—",
  },
];

import { h } from "vue";
</script>

<template>
  <NLayout has-sider style="height: 100vh">
    <NLayoutSider bordered :width="280" content-style="padding: 16px">
      <NSpace vertical size="large">
        <NText strong>Filter</NText>
        <NForm size="small" label-placement="top">
          <NFormItem label="Jenis">
            <NSelect v-model:value="filterJenis" :options="jenisOptions" />
          </NFormItem>
          <NFormItem label="Cari Perihal">
            <NInput v-model:value="filterSearch" placeholder="Kata kunci" clearable />
          </NFormItem>
          <NFormItem label="Tanggal Surat">
            <NDatePicker v-model:value="filterTanggalRange" type="daterange" clearable />
          </NFormItem>
          <NFormItem>
            <NButton type="primary" block @click="applyFilter">Terapkan</NButton>
          </NFormItem>
        </NForm>
      </NSpace>
    </NLayoutSider>

    <NLayout>
      <NLayoutHeader bordered style="padding: 12px 24px">
        <NSpace justify="space-between" align="center">
          <NSpace align="center">
            <NText strong>Daftar Surat</NText>
            <NButton
              size="small"
              tertiary
              @click="router.push({ name: 'inbox' })"
              data-testid="nav-inbox"
            >
              Inbox
            </NButton>
            <NButton
              v-if="auth.hasRole('camat') || auth.hasRole('admin')"
              size="small"
              tertiary
              @click="router.push({ name: 'dashboard' })"
              data-testid="nav-dashboard"
            >
              Dashboard
            </NButton>
            <NButton
              v-if="auth.hasRole('camat') || auth.hasRole('admin')"
              size="small"
              tertiary
              @click="router.push({ name: 'stats' })"
              data-testid="nav-stats"
            >
              Statistik
            </NButton>
            <NButton
              size="small"
              tertiary
              @click="router.push({ name: 'reconciliation' })"
              data-testid="nav-reconciliation"
            >
              Rekonsiliasi
            </NButton>
          </NSpace>
          <NSpace align="center">
            <NButton type="primary" size="small" @click="router.push({ name: 'surat-baru' })">
              + Surat Baru
            </NButton>
            <NButton
              v-if="auth.hasRole('student')"
              size="small"
              tertiary
              :type="eduPanel.enabled ? 'info' : 'default'"
              @click="eduPanel.enabled = !eduPanel.enabled; eduPanel.drawerOpen = eduPanel.enabled"
              data-testid="student-mode-toggle"
              :title="eduPanel.enabled ? 'Tutup panel edukasi' : 'Buka panel edukasi'"
            >
              🎓 {{ eduPanel.enabled ? "ON" : "OFF" }}
            </NButton>
            <PendingSyncIndicator />
            <NotificationBell />
            <NText depth="3">{{ auth.userID }} ({{ auth.roles.join(", ") }})</NText>
            <NButton size="small" @click="themeStore.toggle()">
              {{ themeStore.dark ? "☀" : "☾" }}
            </NButton>
            <NButton size="small" @click="logout">Keluar</NButton>
          </NSpace>
        </NSpace>
      </NLayoutHeader>

      <NLayoutContent style="padding: 24px">
        <NSpin :show="loading && items.length === 0">
          <NEmpty v-if="!loading && items.length === 0" description="Tidak ada surat" />
          <NDataTable
            v-else
            :columns="columns"
            :data="items"
            :row-key="(row: SuratListItem) => row.id"
            :row-props="(row: SuratListItem) => ({
              style: 'cursor: pointer',
              onClick: () => router.push({ name: 'surat-detail', params: { id: row.id } }),
            })"
            :bordered="false"
          />
          <NSpace justify="center" style="margin-top: 16px" v-if="hasMore">
            <NButton :loading="loading" @click="loadMore">Muat lebih banyak</NButton>
          </NSpace>
        </NSpin>
      </NLayoutContent>
    </NLayout>
  </NLayout>
</template>
