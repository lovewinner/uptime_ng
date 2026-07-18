<script setup lang="ts">
import { computed, ref } from 'vue'

const props = defineProps<{
  modelValue: boolean
  monitors: { id: number; name: string; type: string }[]
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', val: boolean): void
  (e: 'export', ids?: number[]): void
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (val) => emit('update:modelValue', val),
})

const chosen = ref<'all' | 'selected'>('all')
const selectedIds = computed(() => props.monitors.filter((m) => m.name !== '').map((m) => m.id))

function confirmExport() {
  emit('export', chosen.value === 'selected' ? selectedIds.value : undefined)
  visible.value = false
}
</script>

<template>
  <el-dialog v-model="visible" title="导出监控配置" width="min(500px, 95%)">
    <p>选择导出范围：</p>
    <el-radio-group v-model="chosen">
      <el-radio value="all">导出全部监控项 ({{ monitors.length }})</el-radio>
      <el-radio value="selected">导出当前列表中的监控项</el-radio>
    </el-radio-group>

    <template #footer>
      <el-button @click="visible = false">取消</el-button>
      <el-button type="primary" @click="confirmExport">导出</el-button>
    </template>
  </el-dialog>
</template>
