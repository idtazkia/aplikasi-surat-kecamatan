<script setup lang="ts">
import { computed } from "vue";
import { NDrawer, NDrawerContent, NEmpty, NCode, NText, NTag, NSpace, NDivider } from "naive-ui";
import { useEduPanelStore } from "@/stores/eduPanel";

defineProps<{
  open: boolean;
}>();

defineEmits<{
  (e: "close"): void;
}>();

const eduPanel = useEduPanelStore();
const payload = computed(() => eduPanel.lastPayload);
const links = computed(() => {
  const out: { id: string; permalink: string; label: string }[] = [];
  for (const id of payload.value?.concept_ids ?? []) {
    const link = eduPanel.linkByID.get(id);
    if (link) {
      out.push({
        id,
        permalink: link.permalink,
        label: `${link.file}:${link.start_line}-${link.end_line}`,
      });
    }
  }
  return out;
});
</script>

<template>
  <NDrawer
    :show="open"
    :width="480"
    placement="right"
    @update:show="(v: boolean) => !v && $emit('close')"
  >
    <NDrawerContent title="Student Mode" closable>
      <NEmpty v-if="!payload" description="Belum ada operasi yang ter-record. Klik link/aksi untuk lihat detail data structure & complexity." />
      <NSpace v-else vertical size="large">
        <div>
          <NText strong>Operation</NText>
          <div>{{ payload.operation }}</div>
        </div>

        <div v-if="payload.data_structures?.length">
          <NText strong>Data Structures</NText>
          <NSpace>
            <NTag v-for="ds in payload.data_structures" :key="ds" type="info">{{ ds }}</NTag>
          </NSpace>
        </div>

        <div v-if="payload.complexity">
          <NText strong>Complexity</NText>
          <div>Theoretical: <code>{{ payload.complexity.theoretical }}</code></div>
          <div v-if="payload.complexity.without_index">
            Without index: <code>{{ payload.complexity.without_index }}</code>
          </div>
        </div>

        <div v-if="payload.sql">
          <NText strong>SQL</NText>
          <NCode :code="payload.sql" language="sql" word-wrap />
        </div>

        <div v-if="payload.explain">
          <NText strong>EXPLAIN ANALYZE</NText>
          <NCode :code="payload.explain" word-wrap />
        </div>

        <div v-if="links.length">
          <NDivider />
          <NText strong>Source Code</NText>
          <NSpace vertical size="small">
            <a v-for="link in links" :key="link.id" :href="link.permalink" target="_blank" rel="noopener">
              {{ link.label }}
            </a>
          </NSpace>
        </div>
      </NSpace>
    </NDrawerContent>
  </NDrawer>
</template>
