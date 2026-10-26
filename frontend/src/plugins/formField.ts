import { computed, type WritableComputedRef } from 'vue'

// The two shapes that most of the writable computeds in the forms collapse to.
// Deliberately narrow: multi-field toggles, unit conversion (connect_timeout's
// 's' suffix, fragment_fallback_delay's 'ms'), and anything whose getter checks
// more than one key stay hand-written.

// owner is a getter, not the object, because the entity is replaced underneath a
// mounted form: DnsServerDrawer's serverType setter does
// `server.value = createDnsServer(...)`, OutboundDrawer and InboundDrawer do the
// same on protocol change, and out/OutTLS's tlsEnable assigns a whole new
// `props.outbound.tls`. MDrawer only re-keys its body when `open` flips, so the
// child stays mounted across those swaps. A reference captured at setup would
// keep writing into the discarded object.
//
// The getters below optional-chain owner() and the setters deliberately do not:
// every call site either owns the object outright or renders behind a v-if on it
// (InboundDrawer's `v-if="inbound.out_json"`), so a setter is only reachable once
// it exists. Same split as out/OutTLS.vue. Adding `?.` there would turn a future
// null owner into a silently dropped write instead of a throw.

/**
 * Presence toggle: on writes the default, off removes the key.
 *
 * `clear` has no default on purpose. `delete o[k]` and `o[k] = undefined` are in
 * fact indistinguishable here — FindDiff.deepCompare filters undefined-valued
 * keys before comparing, JSON.stringify drops them, and none of the Object.hasOwn
 * readers look at these sub-fields (they check entity-level tls/transport/multiplex).
 * But the existing code mixes both spellings, sometimes within one file, and this
 * helper is meant to preserve each site as written rather than quietly pick one.
 */
export function optionField<T>(
  owner: () => any,
  key: string,
  onValue: () => T,
  clear: 'delete' | 'undefined',
): WritableComputedRef<boolean> {
  return computed({
    // Loose `!= undefined`, so a default of '' or 0 still reads as present.
    // Tailscale.vue uses strict `!==` throughout — don't convert it with this.
    get: () => owner()?.[key] != undefined,
    set: (v: boolean) => {
      const o = owner()
      if (v) o[key] = onValue()
      else if (clear === 'delete') delete o[key]
      else o[key] = undefined
    },
  })
}

/**
 * Fallback read, plain write.
 *
 * The setter never writes undefined even for falsy values: several of these
 * fields double as the presence flag for an optionField above them
 * (tcp_fast_open, udp_fragment, reuse_addr, bind_address_no_port,
 * disable_tcp_keep_alive). Clearing the key on `false` would make the toggle
 * switch itself off and take the control out of the DOM.
 */
export function valueField<T>(owner: () => any, key: string, fallback: T): WritableComputedRef<T> {
  return computed({
    get: () => owner()?.[key] ?? fallback,
    set: (v: T) => { owner()[key] = v },
  })
}

/**
 * Optional text: trimmed on write, and a blank box removes the key rather than
 * storing "". sing-box refuses an empty string for a byte size or a duration
 * (memory_limit, min_interval), so leaving one behind fails the whole config.
 */
export function textField(owner: () => any, key: string): WritableComputedRef<string> {
  return computed({
    get: () => owner()?.[key] ?? '',
    set: (v: string) => {
      const o = owner()
      const trimmed = (v ?? '').trim()
      if (trimmed) o[key] = trimmed
      else delete o[key]
    },
  })
}
