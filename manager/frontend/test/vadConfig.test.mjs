import test from 'node:test'
import assert from 'node:assert/strict'
import { mergeVadConfig } from '../src/utils/vad.js'

test('editing a legacy high-threshold config preserves single-threshold behavior', () => {
  const defaults = { threshold: 0.5, threshold_low: 0.35, pool_size: 4 }
  const old = { threshold: 0.8, pool_size: 2 }
  assert.deepEqual(mergeVadConfig(defaults, old), { threshold: 0.8, threshold_low: 0.8, pool_size: 2 })
  assert.equal(defaults.threshold_low, 0.35)
  assert.equal(old.threshold_low, undefined)
})

test('a missing threshold uses the provider default without silently enabling hysteresis', () => {
  assert.deepEqual(mergeVadConfig({ threshold: 0.3, threshold_low: 0.2 }, {}), {
    threshold: 0.3, threshold_low: 0.3
  })
})

test('an explicitly configured zero low threshold is preserved', () => {
  assert.deepEqual(mergeVadConfig({ threshold: 0.5 }, { threshold: 0.8, threshold_low: 0 }), {
    threshold: 0.8, threshold_low: 0
  })
})
