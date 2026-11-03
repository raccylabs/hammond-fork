<script setup>
import { ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { post, auth } from '../lib/api.js'
import { store, setTheme, loadSession } from '../lib/store.js'
import Brand from '../components/Brand.vue'

const router = useRouter()
const route = useRoute()
const email = ref('')
const password = ref('')
const remember = ref(true)
const busy = ref(false)
const err = ref('')

async function submit() {
  err.value = ''
  busy.value = true
  try {
    const r = await post('/login', { email: email.value.trim(), password: password.value })
    auth.set(r.token, remember.value ? r.refreshToken : '')
    await loadSession()
    router.replace(route.query.next || '/garage')
  } catch (x) {
    err.value = x.status === 403 ? x.message.replace(/^login:?\s*/i, '') || 'Email or password is wrong' : x.message
  } finally { busy.value = false }
}
</script>

<template>
  <div class="login">
    <div class="login-art">
      <Brand kind="mark" :width="360" color="#231F20" class="mk" />
      <svg viewBox="0 0 1000 600" aria-hidden="true" preserveAspectRatio="none" class="roads">
        <path pathLength="1" d="M-20,620 C120,420 380,330 1020,300" />
        <path pathLength="1" d="M-20,520 C160,360 420,260 1020,230" style="animation-delay: 0.6s; opacity: 0.6" />
      </svg>
      <div class="tag">Open-sourced twists and turns.</div>
    </div>
    <div class="login-form">
      <form class="stack" style="width: 100%; max-width: 380px; gap: 18px" @submit.prevent="submit">
        <Brand :width="240" class="rise" />
        <div class="rise disp" style="font-size: 22px; font-weight: 600; color: var(--accentText); animation-delay: 0.08s">Welcome back</div>
        <label class="field rise" style="animation-delay: 0.14s">Email address<input v-model="email" type="email" autocomplete="email" required /></label>
        <label class="field rise" style="animation-delay: 0.2s">Password<input v-model="password" type="password" autocomplete="current-password" required /></label>
        <div class="between rise" style="font-size: 14px; animation-delay: 0.26s">
          <button type="button" class="row muted" style="all: unset; display: flex; align-items: center; gap: 10px; cursor: pointer; min-height: 44px" role="switch" :aria-checked="remember" @click="remember = !remember">
            <span class="toggle" :aria-checked="remember"><span></span></span>Stay signed in
          </button>
        </div>
        <div v-if="err" class="error" role="alert">{{ err }}</div>
        <button type="submit" class="btn pri rise" style="min-height: 56px; font-size: 17px; animation-delay: 0.32s" :disabled="busy">{{ busy ? 'Signing in…' : 'Log in' }}</button>
        <div class="between rise label" style="padding-top: 8px; border-top: 1px solid var(--line); animation-delay: 0.4s">
          <span>Self-hosted Hammond</span>
          <button type="button" class="chip" @click="setTheme(store.theme === 'dark' ? 'light' : 'dark')">{{ store.theme === 'dark' ? 'Light mode' : 'Dark mode' }}</button>
        </div>
      </form>
    </div>
  </div>
</template>

<style scoped>
.login { min-height: 100vh; display: flex; flex-wrap: wrap; }
.login-art { flex: 1 1 480px; min-height: 300px; background: #fe4611; position: relative; overflow: hidden; display: flex; align-items: center; justify-content: center; padding: 48px; }
.login-art .mk { width: min(62%, 420px) !important; height: auto !important; animation: mk 1.4s cubic-bezier(0.2, 0.7, 0.2, 1) both; }
.roads { position: absolute; inset: 0; width: 100%; height: 100%; pointer-events: none; }
.roads path { fill: none; stroke: rgba(35, 31, 32, 0.22); stroke-width: 2; stroke-dasharray: 1; animation: draw 2.4s 0.3s cubic-bezier(0.5, 0, 0.2, 1) both; }
.tag { position: absolute; left: 40px; bottom: 32px; font-size: 14px; color: #231f20; font-weight: 600; font-stretch: 112%; }
.login-form { flex: 1 1 420px; display: flex; align-items: center; justify-content: center; padding: 48px 24px; }
.rise { animation: rise 0.7s cubic-bezier(0.2, 0.7, 0.2, 1) both; }
@keyframes mk { from { opacity: 0; transform: translateY(30px) scale(0.96); } }
</style>
