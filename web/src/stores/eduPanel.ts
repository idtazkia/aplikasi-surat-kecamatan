import { defineStore } from "pinia";
import { ref, computed } from "vue";

// Concept-links artifact yang di-emit oleh tools/concept-links emit-json.
// Bundled dengan app build (di-import via fetch saat init), bukan fetch
// runtime, supaya offline-friendly.
export interface ConceptLink {
  id: string;
  permalink: string;
  file: string;
  start_line: number;
  end_line: number;
}

export interface ConceptLinks {
  generated_from_commit: string;
  repo_slug: string;
  links: ConceptLink[];
}

// EduPayload adalah blok `_edu` yang server append ke response saat student
// mode aktif. Frontend render di student drawer.
export interface EduPayload {
  operation: string;
  data_structures?: string[];
  complexity?: {
    theoretical: string;
    without_index?: string;
    actual?: Record<string, unknown>;
  };
  sql?: string;
  explain?: string;
  concept_ids?: string[];
}

export const useEduPanelStore = defineStore("eduPanel", () => {
  // Enabled = student mode aktif untuk user ini.
  // Default off; admin/dev toggle, atau auto-on untuk role 'student'.
  const enabled = ref<boolean>(false);
  const drawerOpen = ref<boolean>(false);
  const lastPayload = ref<EduPayload | null>(null);
  const links = ref<ConceptLinks | null>(null);

  async function loadLinks() {
    // Concept-links.json di-emit ke public/ saat build.
    try {
      const resp = await fetch("/concept-links.json");
      if (resp.ok) {
        links.value = (await resp.json()) as ConceptLinks;
      }
    } catch {
      // Silent fail — student mode masih bisa render payload tanpa permalink.
    }
  }

  function recordPayload(payload: EduPayload) {
    if (!enabled.value) return;
    lastPayload.value = payload;
  }

  // concept:hash-table-map:start
  // Hash table lookup concept_id -> ConceptLink. JavaScript Map = hash table
  // dengan amortized O(1) get/set. Tanpa map, lookup linear O(n) per render
  // student drawer yang punya banyak concept_id (mahal kalau catalog tumbuh).
  // Tradeoff: O(n) build sekali vs O(n*m) per query untuk m render.
  const linkByID = computed(() => {
    const map = new Map<string, ConceptLink>();
    for (const link of links.value?.links ?? []) {
      map.set(link.id, link);
    }
    return map;
  });
  // concept:hash-table-map:end

  return {
    enabled,
    drawerOpen,
    lastPayload,
    links,
    linkByID,
    loadLinks,
    recordPayload,
  };
});
