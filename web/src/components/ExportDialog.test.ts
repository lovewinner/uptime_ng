import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import ExportDialog from './ExportDialog.vue'

describe('ExportDialog', () => {
  it('emits selected ids when selected range is chosen', async () => {
    const wrapper = mount(ExportDialog, {
      props: {
        modelValue: true,
        monitors: [
          { id: 1, name: 'site', type: 'http' },
          { id: 2, name: '', type: 'group' },
        ],
      },
      global: {
        stubs: {
          'el-dialog': { template: '<div><slot /><slot name="footer" /></div>' },
          'el-radio-group': {
            props: ['modelValue'],
            emits: ['update:modelValue'],
            template: '<div><button class="choose-selected" @click="$emit(\'update:modelValue\', \'selected\')"></button><slot /></div>',
          },
          'el-radio': { template: '<label><slot /></label>' },
          'el-button': { template: '<button @click="$emit(\'click\')"><slot /></button>' },
        },
      },
    })

    await wrapper.find('.choose-selected').trigger('click')
    await wrapper.findAll('button').at(-1)!.trigger('click')

    expect(wrapper.emitted('export')?.[0]).toEqual([[1]])
  })
})
