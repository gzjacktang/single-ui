import { getConfig } from '@/api/config'

export const configStore = reactive({
  Username: '',
})

export async function loadConfig() {
  const res = await getConfig()
  configStore.Username = res.Username
}
