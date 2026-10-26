<template>
  <div>
    <SectionLabel style="margin-bottom: 12px;">OOM Killer</SectionLabel>
    <div class="grid2">
      <Field :label="$t('types.oomKiller.memoryLimit')">
        <input class="input mono" v-model="memoryLimit" placeholder="1gb" />
      </Field>
      <Field :label="$t('types.oomKiller.safetyMargin')">
        <input class="input mono" v-model="safetyMargin" placeholder="128mb" />
      </Field>
    </div>
    <div class="grid2">
      <Field :label="$t('types.oomKiller.minInterval')">
        <input class="input mono" v-model="minInterval" placeholder="10s" />
      </Field>
      <Field :label="$t('types.oomKiller.maxInterval')">
        <input class="input mono" v-model="maxInterval" placeholder="1m" />
      </Field>
    </div>
  </div>
</template>

<script lang="ts" setup>
import Field from '@/components/ui/Field.vue'
import SectionLabel from '@/components/ui/SectionLabel.vue'
import { textField } from '@/plugins/formField'

const props = defineProps<{ data: any }>()

// Every field is optional and cleared rather than stored empty.
const owner = () => props.data
const memoryLimit = textField(owner, 'memory_limit')
const safetyMargin = textField(owner, 'safety_margin')
const minInterval = textField(owner, 'min_interval')
const maxInterval = textField(owner, 'max_interval')
</script>
