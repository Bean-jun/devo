<script setup lang="ts">
import { useAppIcon, type IconName } from './AppIconController'

const props = withDefaults(defineProps<{
  name: IconName
  size?: number | string
  weight?: 'thin' | 'light' | 'regular' | 'bold' | 'fill' | 'duotone'
  color?: string
}>(), {
  size: 18,
  weight: 'regular',
})

const { resolvedIcon, iconSize } = useAppIcon(props)
</script>

<template>
  <component
    :is="resolvedIcon"
    :size="iconSize"
    :weight="weight"
    :color="color"
    :style="typeof size === 'string' ? { width: size, height: size } : undefined"
  />
</template>