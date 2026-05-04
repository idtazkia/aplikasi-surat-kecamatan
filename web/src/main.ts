import { createApp } from "vue";
import { createPinia } from "pinia";
import { VueQueryPlugin } from "@tanstack/vue-query";
import App from "./App.vue";
import router from "./router";
import { useOfflineStore } from "@/stores/offline";
import { useAuthStore } from "@/stores/auth";

import "vfonts/Lato.css";
import "vfonts/FiraCode.css";

const app = createApp(App);
const pinia = createPinia();
app.use(pinia);
app.use(router);
app.use(VueQueryPlugin);

// Initialize offline store — bind browser online/offline listener + trigger
// initial sync kalau user sudah login.
const offline = useOfflineStore(pinia);
offline.bindBrowserEvents();
void offline.refreshMetaFromCache();

const auth = useAuthStore(pinia);
if (auth.isAuthenticated && offline.online) {
  void offline.sync();
}

app.mount("#app");
