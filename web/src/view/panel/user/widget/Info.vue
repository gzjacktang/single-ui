<template>
  <div class="info">
    <FormRow title="启用">
      <span class="tag" :class="user.Enable ? 'green' : 'red'">{{
        user.Enable ? '开启' : '关闭'
      }}</span>
    </FormRow>
    <FormRow title="名称">
      <span>{{ user.Name || '-' }}</span>
    </FormRow>
    <FormRow title="UUID">
      <span class="mono">{{ user.UUID }}</span>
      <Copy :value="user.UUID" size="sm" />
    </FormRow>
    <FormRow title="认证">
      <span class="mono">{{ user.Password }}</span>
      <Copy :value="user.Password" size="sm" />
    </FormRow>
    <FormRow title="创建时间">
      <span>{{ formatTime(user.CreatedAt) }}</span>
    </FormRow>
    <FormRow title="更新时间">
      <span>{{ formatTime(user.UpdatedAt) }}</span>
    </FormRow>
    <FormRow title="关联入站">
      <div class="tags">
        <span v-for="ib in inboundTags" :key="ib.id" class="tag" :class="ib.color">
          {{ ib.port }}
        </span>
      </div>
    </FormRow>
    <div class="divider">节点分享</div>
    <Link
      v-for="uri in uris"
      :key="uri.value"
      :label="uri.label"
      :color="uri.color"
      :name="uri.name"
      :value="uri.value"
    />
  </div>
</template>

<script setup lang="ts">
import FormRow from '@/component/ui/FormRow.vue'
import Copy from '@/component/widget/Copy.vue'
import Link from '@/component/widget/Link.vue'
import { formatTime } from '@/util/format'
import { getUri } from '@/api/share'
import { protocol } from '@/util/tag'

const props = defineProps<{
  user: any
  inbounds: any[]
}>()

const inboundTags = computed(() => {
  try {
    const ids: number[] = JSON.parse(props.user.Inbounds || '[]')
    return ids
      .map((id) => {
        const ib = props.inbounds.find((i) => i.ID === id)
        return ib
          ? {
              id,
              port: ib.Port,
              name: ib.Name,
              protocol: ib.Protocol,
              color: protocol(ib.Protocol),
              inbound: ib,
            }
          : null
      })
      .filter(
        (
          ib,
        ): ib is {
          id: number
          port: any
          name: string
          protocol: string
          color: string
          inbound: any
        } => ib !== null,
      )
  } catch {
    return []
  }
})

const uris = ref<
  { label: string; color: string; name: string; value: string; download?: string }[]
>([])
let linkLoadVersion = 0
async function loadLinks() {
  const version = ++linkLoadVersion
  const user = { ...props.user }
  const tags = [...inboundTags.value]
  const uriResults = await Promise.all(tags.map((ib) => getUri({ user, inbound: ib.inbound })))
  if (version !== linkLoadVersion) return

  uris.value = tags.map((ib, i) => ({
    label: ib.protocol.toUpperCase(),
    color: ib.color,
    name: ib.name,
    value: uriResults[i].uri,
  }))
}

watch([() => props.user, () => props.inbounds], loadLinks, { immediate: true, deep: true })
</script>

<style scoped>
.info {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 20px;
}

.mono {
  font-family: monospace;
  font-size: var(--font-size-sm);
  color: var(--color-text-dark);
}
</style>
