<template>
  <div v-if="ech" class="fade-up">
    <div style="display: flex; align-items: center; gap: 10px; margin-bottom: 12px;">
      <Segmented
        v-model="useEchPath"
        :options="[[0, $t('tls.usePath')], [1, $t('tls.useText')]]"
        @update:model-value="switchEchMode"
      />
      <div style="flex: 1;" />
      <Btn variant="subtle" sm :loading="loading" :title="$t('actions.generate')" @click="genECH">
        <Ico name="key" :size="14" /> {{ $t('actions.generate') }}
      </Btn>
    </div>

    <Field v-if="useEchPath == 0" :label="$t('tls.keyPath')">
      <input class="input mono" v-model="ech.key_path" />
    </Field>
    <Field v-else :label="$t('tls.key')">
      <textarea class="input mono" rows="4" spellcheck="false" v-model="echKeyText"></textarea>
    </Field>

    <Field v-if="oTls.ech" :label="$t('tls.queryServerName')">
      <input class="input mono" v-model="oTls.ech.query_server_name" placeholder="ech.example.com" />
    </Field>

    <Field :label="$t('tls.cert')">
      <textarea class="input mono" rows="4" spellcheck="false" v-model="echConfigText"></textarea>
    </Field>
  </div>
</template>

<script lang="ts" setup>
import { computed, ref } from 'vue'
import HttpUtils from '@/plugins/httputil'
import { ech as echType } from '@/types/tls'
import Field from '@/components/ui/Field.vue'
import Btn from '@/components/ui/Btn.vue'
import Ico from '@/components/ui/Ico.vue'
import Segmented from '@/components/ui/Segmented.vue'

// requireSni is the drawer's: it owns the SNI field, which it switches on and
// focuses when the name ECH is generated for is still empty.
const props = defineProps<{ iTls: any; oTls: any; requireSni: () => boolean }>()

const useEchPath = ref<string | number>(props.iTls?.ech?.key ? 1 : 0)
const loading = ref(false)

const ech = computed((): echType => props.iTls.ech)

const switchEchMode = (v: string | number) => {
  if (v == 0) delete props.iTls.ech.key
  else delete props.iTls.ech.key_path
}

const echKeyText = computed({
  get: (): string => (ech.value?.key ? ech.value.key.join('\n') : ''),
  set: (v: string) => { ech.value.key = v.split('\n') },
})
const echConfigText = computed({
  get: (): string => (props.oTls.ech?.config ? props.oTls.ech.config.join('\n') : ''),
  set: (v: string) => { props.oTls.ech.config = v.split('\n') },
})

const genECH = async () => {
  if (!props.requireSni()) return
  loading.value = true
  const msg = await HttpUtils.get('api/keypairs', {
    k: 'ech',
    // The ECH public name; the panel refuses one that is not a domain.
    o: props.iTls.server_name,
  })
  loading.value = false
  // A failure is success:false, already shown by HttpUtils; success has lines.
  if (msg.success && props.iTls.ech && props.oTls.ech) {
    props.iTls.ech.key_path = undefined
    useEchPath.value = 1
    const config = <string[]>[]
    const key = <string[]>[]
    let isConfig = false
    let isKey = false

    msg.obj.forEach((line: string) => {
      if (line === '-----BEGIN ECH CONFIGS-----') {
        isConfig = true
        isKey = false
        config.push(line)
      } else if (line === '-----END ECH CONFIGS-----') {
        isConfig = false
        config.push(line)
      } else if (line === '-----BEGIN ECH KEYS-----') {
        isKey = true
        isConfig = false
        key.push(line)
      } else if (line === '-----END ECH KEYS-----') {
        isKey = false
        key.push(line)
      } else if (isConfig) {
        config.push(line)
      } else if (isKey) {
        key.push(line)
      }
    })
    props.iTls.ech.key = key
    props.oTls.ech.config = config
  }
}
</script>
