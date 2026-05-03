<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from "vue";
import { useRouter } from "vue-router";
import {
  NButton, NBadge, NPopover, NSpace, NText, NList, NListItem,
  NThing, NTag, NEmpty, NSpin, useMessage,
} from "naive-ui";
import { notificationApi, type NotificationItem } from "@/api/surat";

const router = useRouter();
const message = useMessage();

const items = ref<NotificationItem[]>([]);
const unread = ref(0);
const loading = ref(false);
let pollHandle: ReturnType<typeof setInterval> | null = null;

const POLL_INTERVAL_MS = 30_000; // poll setiap 30s — Fase 7 ganti push

async function fetch() {
  loading.value = true;
  try {
    const resp = await notificationApi.list(false);
    items.value = resp.items;
    unread.value = resp.unread;
  } catch (e) {
    console.error("notif fetch:", e);
  } finally {
    loading.value = false;
  }
}

async function markRead(n: NotificationItem) {
  try {
    await notificationApi.markRead(n.id);
    await fetch();
  } catch (e) {
    message.error("Gagal mark as read");
    console.error(e);
  }
}

async function markAllRead() {
  try {
    await notificationApi.markAllRead();
    await fetch();
    message.success("Semua notifikasi ditandai dibaca");
  } catch (e) {
    message.error("Gagal mark all read");
    console.error(e);
  }
}

async function openTarget(n: NotificationItem) {
  // Auto mark read saat klik
  if (!n.read_at) {
    await notificationApi.markRead(n.id);
  }
  const suratID = n.payload.surat_id as string | undefined;
  if (suratID) {
    router.push({ name: "surat-detail", params: { id: suratID } });
  }
}

function notifTitle(n: NotificationItem): string {
  const p = n.payload as { surat_perihal?: string; surat_nomor?: string };
  const perihal = p.surat_perihal ?? "(surat tidak terdeskripsi)";
  if (n.type === "disposisi_baru") return `Disposisi baru: ${perihal}`;
  if (n.type === "komentar_baru") return `Komentar baru di: ${perihal}`;
  return n.type;
}

function notifDescription(n: NotificationItem): string {
  const p = n.payload as { instruksi?: string; body_excerpt?: string };
  if (n.type === "disposisi_baru") return p.instruksi ?? "";
  if (n.type === "komentar_baru") return p.body_excerpt ?? "";
  return "";
}

function notifTime(n: NotificationItem): string {
  return new Date(n.created_at).toLocaleString("id-ID", {
    month: "short", day: "numeric", hour: "2-digit", minute: "2-digit",
  });
}

const hasUnread = computed(() => unread.value > 0);

onMounted(() => {
  fetch();
  pollHandle = setInterval(fetch, POLL_INTERVAL_MS);
});

onUnmounted(() => {
  if (pollHandle) clearInterval(pollHandle);
});
</script>

<template>
  <NPopover trigger="click" :width="380" placement="bottom-end">
    <template #trigger>
      <NBadge :value="unread" :max="99" :show="hasUnread" data-testid="notif-badge">
        <NButton size="small" tertiary data-testid="notif-bell">
          🔔
        </NButton>
      </NBadge>
    </template>
    <NSpace vertical size="small" style="max-height: 400px; overflow-y: auto">
      <NSpace justify="space-between" align="center">
        <NText strong>Notifikasi</NText>
        <NButton
          v-if="hasUnread"
          size="tiny"
          tertiary
          @click="markAllRead"
          data-testid="notif-mark-all-read"
        >
          Tandai semua dibaca
        </NButton>
      </NSpace>
      <NSpin :show="loading">
        <NEmpty v-if="!loading && items.length === 0" description="Tidak ada notifikasi" size="small" />
        <NList v-else>
          <NListItem
            v-for="n in items"
            :key="n.id"
            style="cursor: pointer"
            :data-testid="`notif-item-${n.id}`"
            @click="openTarget(n)"
          >
            <NThing>
              <template #header>
                <NSpace align="center" :size="6">
                  <NTag v-if="!n.read_at" size="tiny" type="info">Baru</NTag>
                  <NText :depth="n.read_at ? 3 : 1" style="font-size: 14px">
                    {{ notifTitle(n) }}
                  </NText>
                </NSpace>
              </template>
              <template #header-extra>
                <NText depth="3" style="font-size: 11px">{{ notifTime(n) }}</NText>
              </template>
              <template #description>
                <NText :depth="n.read_at ? 3 : 2" style="font-size: 12px">
                  {{ notifDescription(n) }}
                </NText>
              </template>
              <template v-if="!n.read_at" #action>
                <NButton size="tiny" tertiary @click.stop="markRead(n)">Tandai dibaca</NButton>
              </template>
            </NThing>
          </NListItem>
        </NList>
      </NSpin>
    </NSpace>
  </NPopover>
</template>
