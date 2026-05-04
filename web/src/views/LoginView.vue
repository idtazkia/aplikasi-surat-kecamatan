<script setup lang="ts">
import { ref } from "vue";
import { useRouter, useRoute } from "vue-router";
import { NCard, NForm, NFormItem, NInput, NButton, NSpace, useMessage } from "naive-ui";
import { useAuthStore } from "@/stores/auth";
import { ApiError } from "@/api/client";

const router = useRouter();
const route = useRoute();
const auth = useAuthStore();
const message = useMessage();

const username = ref("");
const password = ref("");
const submitting = ref(false);

async function handleSubmit() {
  if (!username.value || !password.value) {
    message.warning("Username dan password wajib diisi");
    return;
  }
  submitting.value = true;
  try {
    await auth.login(username.value, password.value);
    const next = (route.query.next as string) ?? "/";
    await router.push(next);
  } catch (err) {
    if (err instanceof ApiError && err.status === 401) {
      message.error("Username atau password salah");
    } else {
      message.error("Login gagal — coba lagi");
    }
  } finally {
    submitting.value = false;
  }
}
</script>

<template>
  <div class="login-wrapper">
    <NCard title="Aplikasi Surat Kecamatan" class="login-card">
      <NForm @submit.prevent="handleSubmit">
        <NFormItem label="Username">
          <NInput v-model:value="username" placeholder="staf1 / camat / admin / auditor" />
        </NFormItem>
        <NFormItem label="Password">
          <NInput
            v-model:value="password"
            type="password"
            show-password-on="click"
            placeholder="demo123"
            @keyup.enter="handleSubmit"
          />
        </NFormItem>
        <NSpace>
          <NButton type="primary" :loading="submitting" @click="handleSubmit">Masuk</NButton>
        </NSpace>
      </NForm>
    </NCard>
  </div>
</template>

<style scoped>
.login-wrapper {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
}
.login-card {
  max-width: 420px;
  width: 100%;
}
</style>
