import { ref, watch } from 'vue'
import { store } from './store.js'
import { get } from './api.js'

/** Loads fillups + expenses for the vehicle returned by `idGetter`, reloading when data changes. */
export function useVehicleData(idGetter) {
  const fillups = ref([])
  const expenses = ref([])
  const loading = ref(true)
  async function load() {
    const id = idGetter()
    if (!id) { fillups.value = []; expenses.value = []; loading.value = false; return }
    loading.value = true
    try {
      const [f, e] = await Promise.all([get(`/vehicles/${id}/fillups`), get(`/vehicles/${id}/expenses`)])
      fillups.value = f || []
      expenses.value = e || []
    } finally { loading.value = false }
  }
  watch([idGetter, () => store.dataVersion], load, { immediate: true })
  return { fillups, expenses, loading, reload: load }
}

/** Loads fillups + expenses for every vehicle. */
export function useAllData() {
  const data = ref({})
  const loading = ref(true)
  async function load() {
    loading.value = true
    const out = {}
    await Promise.all(store.vehicles.map(async (v) => {
      const [f, e] = await Promise.all([get(`/vehicles/${v.id}/fillups`), get(`/vehicles/${v.id}/expenses`)])
      out[v.id] = { fillups: f || [], expenses: e || [] }
    }))
    data.value = out
    loading.value = false
  }
  watch([() => store.vehicles.length, () => store.dataVersion], load, { immediate: true })
  return { data, loading }
}
