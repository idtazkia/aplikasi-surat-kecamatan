import { createRouter, createWebHistory } from "vue-router";
import { useAuthStore } from "@/stores/auth";

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: "/login",
      name: "login",
      component: () => import("@/views/LoginView.vue"),
      meta: { public: true },
    },
    {
      path: "/",
      name: "home",
      redirect: { name: "surat-list" },
    },
    {
      path: "/surat",
      name: "surat-list",
      component: () => import("@/views/SuratListView.vue"),
    },
    {
      path: "/surat/:id",
      name: "surat-detail",
      component: () => import("@/views/SuratDetailView.vue"),
    },
  ],
});

router.beforeEach((to) => {
  const auth = useAuthStore();
  if (to.meta.public) return true;
  if (!auth.accessToken) {
    return { name: "login", query: { next: to.fullPath } };
  }
  return true;
});

export default router;
