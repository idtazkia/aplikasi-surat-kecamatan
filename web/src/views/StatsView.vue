<script setup lang="ts">
import { ref, computed, onMounted, h } from "vue";
import { useRouter } from "vue-router";
import {
  NLayout, NLayoutHeader, NLayoutContent, NSpace, NButton, NText, NCard,
  NSpin, NDataTable, NEmpty, NTag, NProgress, NGrid, NGridItem, useMessage,
} from "naive-ui";
import type { DataTableColumns } from "naive-ui";
import { useAuthStore } from "@/stores/auth";
import { useThemeStore } from "@/stores/theme";
import {
  statsApi,
  type StatsByPeriod,
  type StatsByClassification,
  type StatsBySender,
  type StatsStaffLoad,
} from "@/api/surat";
import NotificationBell from "@/components/NotificationBell.vue";
import PendingSyncIndicator from "@/components/PendingSyncIndicator.vue";

const router = useRouter();
const auth = useAuthStore();
const themeStore = useThemeStore();
const message = useMessage();

const period = ref<StatsByPeriod[]>([]);
const klasifikasi = ref<StatsByClassification[]>([]);
const sender = ref<StatsBySender[]>([]);
const staffLoad = ref<StatsStaffLoad[]>([]);
const loading = ref(false);

async function fetchAll() {
  loading.value = true;
  try {
    const [p, k, s, sl] = await Promise.all([
      statsApi.byPeriod(),
      statsApi.byClassification(),
      statsApi.bySender(10),
      statsApi.staffLoad(),
    ]);
    period.value = p.items;
    klasifikasi.value = k.items;
    sender.value = s.items;
    staffLoad.value = sl.items;
  } catch (e) {
    message.error("Gagal memuat statistik");
    console.error(e);
  } finally {
    loading.value = false;
  }
}

function logout() {
  auth.logout();
  router.push({ name: "login" });
}

// Period bar viz: hitung max untuk normalize bar width
const periodMax = computed(() => {
  let max = 0;
  for (const p of period.value) {
    const total = (p.jenis_count.masuk ?? 0) + (p.jenis_count.keluar ?? 0);
    if (total > max) max = total;
  }
  return max || 1;
});

const senderMax = computed(() =>
  sender.value.length > 0 ? sender.value[0].count : 1,
);

// Klasifikasi table columns
const klasifikasiColumns: DataTableColumns<StatsByClassification> = [
  {
    title: "Kode",
    key: "klasifikasi_kode",
    render: (row) => row.klasifikasi_kode ?? "(tanpa)",
  },
  {
    title: "Nama",
    key: "klasifikasi_nama",
    render: (row) => row.klasifikasi_nama ?? "(tanpa klasifikasi)",
  },
  { title: "Jumlah", key: "count", align: "right" },
];

// Staff load table columns
const staffColumns: DataTableColumns<StatsStaffLoad> = [
  { title: "Nama", key: "full_name" },
  {
    title: "Aktif (pending+dikerjakan)",
    key: "total_active",
    align: "right",
  },
  {
    title: "Overdue",
    key: "overdue_count",
    align: "right",
    render: (row) =>
      row.overdue_count > 0
        ? h(NTag, { type: "error", size: "small" }, () => row.overdue_count.toString())
        : row.overdue_count.toString(),
  },
  {
    title: "Selesai",
    key: "done_count",
    align: "right",
    render: (row) => row.status_count.done ?? 0,
  },
];

onMounted(fetchAll);
</script>

<template>
  <NLayout style="height: 100vh">
    <NLayoutHeader bordered style="padding: 12px 24px">
      <NSpace justify="space-between" align="center">
        <NSpace align="center">
          <NButton text @click="router.push({ name: 'surat-list' })">← Daftar Surat</NButton>
          <NText strong>Statistik</NText>
        </NSpace>
        <NSpace align="center">
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
      <NSpin :show="loading">
        <NGrid :cols="2" :x-gap="16" :y-gap="16">
          <!-- Surat per bulan (time series bars) -->
          <NGridItem :span="2">
            <NCard title="Surat per Bulan" data-testid="stats-period-card">
              <NEmpty v-if="period.length === 0" description="Tidak ada data" size="small" />
              <NSpace v-else vertical :size="6">
                <div
                  v-for="p in period"
                  :key="p.bucket"
                  :data-testid="`period-row-${p.bucket}`"
                >
                  <NSpace align="center" :size="8" style="font-size: 13px">
                    <NText strong style="min-width: 80px">{{ p.bucket }}</NText>
                    <NText depth="3" style="min-width: 100px">
                      Masuk {{ p.jenis_count.masuk ?? 0 }} · Keluar {{ p.jenis_count.keluar ?? 0 }}
                    </NText>
                    <div style="flex: 1; min-width: 200px">
                      <NProgress
                        type="line"
                        :percentage="((p.jenis_count.masuk ?? 0) / periodMax) * 100"
                        :show-indicator="false"
                        color="#52c41a"
                        :height="8"
                      />
                      <NProgress
                        v-if="p.jenis_count.keluar"
                        type="line"
                        :percentage="((p.jenis_count.keluar ?? 0) / periodMax) * 100"
                        :show-indicator="false"
                        color="#1890ff"
                        :height="8"
                        style="margin-top: 2px"
                      />
                    </div>
                  </NSpace>
                </div>
              </NSpace>
            </NCard>
          </NGridItem>

          <!-- Top sender (bars) -->
          <NGridItem>
            <NCard title="Top 10 Pengirim" data-testid="stats-sender-card">
              <NEmpty v-if="sender.length === 0" description="Tidak ada data" size="small" />
              <NSpace v-else vertical :size="6">
                <div v-for="s in sender" :key="s.instansi_id" :data-testid="`sender-row-${s.instansi_id}`">
                  <NSpace align="center" :size="8" style="font-size: 13px">
                    <NText strong style="min-width: 200px; max-width: 200px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap">
                      {{ s.instansi_nama }}
                    </NText>
                    <div style="flex: 1; min-width: 100px">
                      <NProgress
                        type="line"
                        :percentage="(s.count / senderMax) * 100"
                        :show-indicator="false"
                        color="#722ed1"
                        :height="10"
                      />
                    </div>
                    <NText depth="3" style="min-width: 30px; text-align: right">{{ s.count }}</NText>
                  </NSpace>
                </div>
              </NSpace>
            </NCard>
          </NGridItem>

          <!-- Klasifikasi (table) -->
          <NGridItem>
            <NCard title="Per Klasifikasi" data-testid="stats-classification-card">
              <NDataTable
                :columns="klasifikasiColumns"
                :data="klasifikasi"
                :pagination="{ pageSize: 10 }"
                size="small"
                :bordered="false"
              />
            </NCard>
          </NGridItem>

          <!-- Beban staf (table) -->
          <NGridItem :span="2">
            <NCard title="Beban Disposisi per Staf" data-testid="stats-staff-load-card">
              <NDataTable
                :columns="staffColumns"
                :data="staffLoad"
                size="small"
                :bordered="false"
              >
                <template #empty>
                  <NEmpty description="Belum ada disposisi" size="small" />
                </template>
              </NDataTable>
            </NCard>
          </NGridItem>
        </NGrid>
      </NSpin>
    </NLayoutContent>
  </NLayout>
</template>
