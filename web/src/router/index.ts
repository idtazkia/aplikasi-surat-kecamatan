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
      path: "/surat/baru",
      name: "surat-baru",
      component: () => import("@/views/SuratFormView.vue"),
    },
    {
      path: "/surat/:id",
      name: "surat-detail",
      component: () => import("@/views/SuratDetailView.vue"),
    },
    {
      path: "/surat/:id/edit",
      name: "surat-edit",
      component: () => import("@/views/SuratFormView.vue"),
    },
    {
      path: "/inbox",
      name: "inbox",
      component: () => import("@/views/InboxView.vue"),
    },
    {
      path: "/dashboard",
      name: "dashboard",
      component: () => import("@/views/DashboardView.vue"),
      meta: { requireRole: ["camat", "admin"] },
    },
  ],
});

router.beforeEach((to) => {
  const auth = useAuthStore();
  if (to.meta.public) return true;
  if (!auth.accessToken) {
    return { name: "login", query: { next: to.fullPath } };
  }
  const required = to.meta.requireRole as string[] | undefined;
  if (required && !required.some((r) => auth.hasRole(r))) {
    return { name: "surat-list" };
  }
  return true;
});

export default router;
