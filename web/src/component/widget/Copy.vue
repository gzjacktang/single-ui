<template>
  <span class="copy" @click="handle">
    <slot />
    <i class="icon" :class="{ sm: props.size === 'sm' }">{{ copied ? 'check' : 'content_copy' }}</i>
  </span>
</template>

<script setup lang="ts">
const props = defineProps<{
  value: string
  size?: 'sm'
}>()

const copied = ref(false)

async function handle() {
  try {
    if (navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(props.value)
    } else {
      const textarea = document.createElement('textarea')
      textarea.value = props.value
      textarea.setAttribute('readonly', '')
      textarea.style.position = 'fixed'
      textarea.style.opacity = '0'
      document.body.appendChild(textarea)
      textarea.select()
      const copiedByCommand = document.execCommand('copy')
      textarea.remove()
      if (!copiedByCommand) throw new Error('copy command failed')
    }
    copied.value = true
    setTimeout(() => (copied.value = false), 2000)
  } catch {
    copied.value = false
  }
}
</script>

<style scoped>
.copy {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 5px;
  border-radius: 5px;
  cursor: pointer;
  color: var(--color-text-light);

  .icon {
    font-size: 20px;
    color: var(--color-text-dark);
    transition: color 0.3s;

    &.sm {
      font-size: 14px;
    }
  }

  &:hover {
    background-color: var(--color-bg-light);

    .icon {
      color: var(--color-text-light);
    }
  }
}
</style>
