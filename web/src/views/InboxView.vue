<script setup lang="ts">
import { ref, onMounted, computed } from "vue";
import { useRouter } from "vue-router";
import {
  NLayout, NLayoutHeader, NLayoutContent, NSpace, NButton, NText,
  NCard, NList, NListItem, NThing, NTag, NSelect, NSpin, NEmpty, useMessage,
} from "naive-ui";
import { useAuthStore } from "@/stores/auth";
import { useThemeStore } from "@/stores/theme";
import { disposisiApi, type Disposisi, type DisposisiStatus } from "@/api/surat";
import NotificationBell from "@/components/NotificationBell.vue";

const router = useRouter();
const auth = useAuthStore();
const themeStore = useThemeStore();
const message = useMessage();

const items = ref<Disposisi[]>([]);
const loading = ref(false);
const filterStatus = ref<DisposisiStatus | "">("");

const statusOptions: { label: string; value: DisposisiStatus | "" }[] = [
  { label: "Semua", value: "" },
  { label: "Pending", value: "pending" },
  { label: "Sedang dikerjakan", value: "in_progress" },
  { label: "Selesai", value: "done" },
  { label: "Dibatalkan", value: "cancelled" },
];

const statusTagType: Record<DisposisiStatus, "default" | "warning" | "info" | "success" | "error"> = {
  pending: "warning",
  in_progress: "info",
  done: "success",
  cancelled: "error",
};

const statusLabel: Record<DisposisiStatus, string> = {
  pending: "Pending",
  in_progress: "Sedang dikerjakan",
  done: "Selesai",
  cancelled: "Dibatalkan",
};

async function fetch() {
  loading.value = true;
  try {
    const params: Parameters<typeof disposisiApi.list>[0] = { mine: true };
    if (filterStatus.value) params.status = filterStatus.value;
    const resp = await disposisiApi.list(params);
    items.value = resp.items;
  } catch (e) {
    message.error("Gagal memuat inbox");
    console.error(e);
  } finally {
    loading.value = false;
  }
}

function isOverdue(d: Disposisi): boolean {
  if (!d.deadline) return false;
  if (d.status === "done" || d.status === "cancelled") return false;
  return new Date(d.deadline).getTime() < Date.now();
}

function formatDeadline(iso: string): string {
  return new Date(iso).toLocaleString("id-ID", {
    year: "numeric", month: "short", day: "numeric",
    hour: "2-digit", minute: "2-digit",
  });
}

function logout() {
  auth.logout();
  router.push({ name: "login" });
}

function openSurat(suratID: string) {
  router.push({ name: "surat-detail", params: { id: suratID } });
}

const summary = computed(() => {
  const pending = items.value.filter((d) => d.status === "pending").length;
  const inProgress = items.value.filter((d) => d.status === "in_progress").length;
  const overdue = items.value.filter((d) => isOverdue(d)).length;
  return { pending, inProgress, overdue };
});

onMounted(fetch);
</script>

<template>
  <NLayout style="height: 100vh">
    <NLayoutHeader bordered style="padding: 12px 24px">
      <NSpace justify="space-between" align="center">
        <NSpace align="center">
          <NButton text @click="router.push({ name: 'surat-list' })">← Daftar Surat</NButton>
          <NText strong>Inbox: Disposisi Saya</NText>
        </NSpace>
        <NSpace align="center">
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
      <NSpace vertical size="large">
        <!-- Summary stats -->
        <NSpace>
          <NCard size="small" style="min-width: 120px" data-testid="inbox-pending">
            <NText depth="3" style="font-size: 12px">Pending</NText>
            <div style="font-size: 24px; font-weight: 600">{{ summary.pending }}</div>
          </NCard>
          <NCard size="small" style="min-width: 120px" data-testid="inbox-in-progress">
            <NText depth="3" style="font-size: 12px">Dikerjakan</NText>
            <div style="font-size: 24px; font-weight: 600">{{ summary.inProgress }}</div>
          </NCard>
          <NCard size="small" style="min-width: 120px" data-testid="inbox-overdue">
            <NText depth="3" style="font-size: 12px">Overdue</NText>
            <div style="font-size: 24px; font-weight: 600; color: #d03050">
              {{ summary.overdue }}
            </div>
          </NCard>
        </NSpace>

        <!-- Filter -->
        <NSpace align="center">
          <NText>Filter status:</NText>
          <NSelect
            v-model:value="filterStatus"
            :options="statusOptions"
            style="width: 220px"
            @update:value="fetch"
          />
        </NSpace>

        <!-- List -->
        <NSpin :show="loading">
          <NEmpty v-if="!loading && items.length === 0" description="Tidak ada disposisi" />
          <NList v-else>
            <NListItem
              v-for="d in items"
              :key="d.id"
              style="cursor: pointer"
              :data-testid="`inbox-item-${d.id}`"
              @click="openSurat(d.surat_id)"
            >
              <NThing>
                <template #header>
                  <NSpace align="center" :size="8">
                    <NTag size="small" :type="statusTagType[d.status]">
                      {{ statusLabel[d.status] }}
                    </NTag>
                    <NTag v-if="isOverdue(d)" size="tiny" type="error">Overdue</NTag>
                    <NText strong>{{ d.surat_perihal }}</NText>
                  </NSpace>
                </template>
                <template #header-extra>
                  <NText depth="3" style="font-size: 12px">
                    <code>{{ d.surat_nomor }}</code>
                  </NText>
                </template>
                <template #description>
                  <div>{{ d.instruksi }}</div>
                  <div style="margin-top: 4px; font-size: 12px; opacity: 0.7">
                    Dari {{ d.creator_name }}
                    <span v-if="d.deadline"> · Deadline: {{ formatDeadline(d.deadline) }}</span>
                  </div>
                </template>
              </NThing>
            </NListItem>
          </NList>
        </NSpin>
      </NSpace>
    </NLayoutContent>
  </NLayout>
</template>
