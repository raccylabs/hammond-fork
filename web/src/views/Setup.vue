<script setup>
import { ref, reactive } from 'vue'
import { useRouter } from 'vue-router'
import { post, auth } from '../lib/api.js'
import { loadSession } from '../lib/store.js'
import { markInitialized } from '../main.js'
import Brand from '../components/Brand.vue'

const router = useRouter()
const f = reactive({ name: '', email: '', password: '', currency: 'USD', distanceUnit: 0 })
const busy = ref(false)
const err = ref('')

async function submit() {
  err.value = ''
  if (f.password.length < 8) { err.value = 'Use at least 8 characters for the password.'; return }
  busy.value = true
  try {
    await post('/auth/initialize', { ...f, email: f.email.trim().toLowerCase() })
    markInitialized()
    const r = await post('/login', { email: f.email.trim().toLowerCase(), password: f.password })
    auth.set(r.token, r.refreshToken)
    await loadSession()
    router.replace('/garage')
  } catch (x) { err.value = x.message } finally { busy.value = false }
}
</script>
<template>
  <div style="min-height: 100vh; display: flex; align-items: center; justify-content: center; padding: 32px 16px">
    <form class="card stack" style="width: 100%; max-width: 460px; gap: 18px" @submit.prevent="submit">
      <Brand :width="200" />
      <div><h1 class="disp" style="font-size: 24px">Set up Hammond</h1><p class="muted" style="margin: 6px 0 0">Create the admin account for this server. You can invite others afterwards.</p></div>
      <label class="field">Your name<input v-model="f.name" autocomplete="name" /></label>
      <label class="field">Email<input v-model="f.email" type="email" autocomplete="email" required /></label>
      <label class="field">Password<input v-model="f.password" type="password" autocomplete="new-password" required /></label>
      <div class="grid2">
        <label class="field">Currency<input v-model="f.currency" maxlength="3" style="text-transform: uppercase" /></label>
        <label class="field">Distance<select v-model.number="f.distanceUnit"><option :value="0">Miles</option><option :value="1">Kilometers</option></select></label>
      </div>
      <div v-if="err" class="error" role="alert">{{ err }}</div>
      <button class="btn pri" style="min-height: 54px" :disabled="busy">Create account</button>
    </form>
  </div>
</template>
