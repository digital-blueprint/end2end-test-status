import { onMounted, onUnmounted, readonly, ref } from 'vue'

const now = ref(Date.now())
let subscribers = 0
let timer: ReturnType<typeof setInterval> | undefined

// All visible dates share one clock, even on large history pages.
export function useNow() {
  onMounted(() => {
    now.value = Date.now()
    subscribers++
    if (!timer) timer = setInterval(() => { now.value = Date.now() }, 30_000)
  })
  onUnmounted(() => {
    subscribers--
    if (subscribers === 0 && timer) {
      clearInterval(timer)
      timer = undefined
    }
  })
  return readonly(now)
}
