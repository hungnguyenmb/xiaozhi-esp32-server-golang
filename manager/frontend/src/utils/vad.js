// Editing a legacy single-threshold config must not silently enable hysteresis.
export function mergeVadConfig(defaults, config) {
  return {
    ...defaults,
    ...config,
    threshold_low: config.threshold_low ?? config.threshold ?? defaults.threshold
  }
}
