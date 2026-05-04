<script setup lang="ts">
import { onMounted } from "vue";
import {
  NButton, NBadge, NPopover, NSpace, NText, NList, NListItem,
  NThing, NTag, NEmpty,
} from "naive-ui";
import { usePendingOpsStore } from "@/stores/pendingOps";
import { drainQueue } from "@/offline/opqueue";

const pendingOps = usePendingOpsStore();

onMounted(() => {
  pendingOps.start();
});

async function manualDrain() {
  await drainQueue();
}

function actionLabel(action: string): string {
  switch (action) {
    case "create": return "Buat";
    case "update": return "Edit";
    case "delete": return "Hapus";
    case "append": return "Komentar";
    default: return action;
  }
}

function entityLabel(type: string): string {
  return type === "surat" ? "Surat" : type === "komentar" ? "Komentar" : type;
}

function formatTime(iso: string): string {
  return new Date(iso).toLocaleString("id-ID", {
    hour: "2-digit", minute: "2-digit", day: "numeric", month: "short",
  });
}
</script>

<template>
  <NPopover trigger="click" :width="380" placement="bottom-end">
    <template #trigger>
      <NBadge :value="pendingOps.count" :max="99" :show="pendingOps.count > 0" data-testid="pending-sync-badge">
        <NButton size="small" tertiary data-testid="pending-sync-button">
          ⏳
        </NButton>
      </NBadge>
    </template>
    <NSpace vertical size="small" style="max-height: 400px; overflow-y: auto">
      <NSpace justify="space-between" align="center">
        <NText strong>Perubahan Menunggu Sync</NText>
        <NButton
          v-if="pendingOps.count > 0"
          size="tiny"
          tertiary
          @click="manualDrain"
          data-testid="manual-drain-btn"
        >
          Sync sekarang
        </NButton>
      </NSpace>
      <NEmpty v-if="pendingOps.count === 0" description="Semua perubahan tersinkron" size="small" />
      <NList v-else>
        <NListItem v-for="op in pendingOps.items" :key="op.client_op_id">
          <NThing>
            <template #header>
              <NSpace align="center" :size="6">
                <NTag size="tiny" :type="op.error ? 'error' : 'warning'">
                  {{ actionLabel(op.action) }}
                </NTag>
                <NText style="font-size: 13px">
                  {{ entityLabel(op.entity_type) }}
                </NText>
              </NSpace>
            </template>
            <template #header-extra>
              <NText depth="3" style="font-size: 11px">
                {{ formatTime(op.client_timestamp) }}
              </NText>
            </template>
            <template #description>
              <NText
                v-if="op.error"
                depth="3"
                style="font-size: 11px; color: #d03050"
              >
                Error: {{ op.error }}
                <span v-if="op.retry_count > 0"> · retry #{{ op.retry_count }}</span>
              </NText>
              <NText v-else depth="3" style="font-size: 11px">
                Menunggu koneksi...
              </NText>
            </template>
          </NThing>
        </NListItem>
      </NList>
    </NSpace>
  </NPopover>
</template>
