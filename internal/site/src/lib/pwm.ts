import type { SystemStatsRecord } from "@/types"

/** Missing or invalid samples remain gaps, while a stopped output is a valid 0%. */
export function pwmValue(record: SystemStatsRecord, sensor: string): number | undefined {
	const value = record.stats?.pwm?.[sensor]
	return typeof value === "number" && Number.isFinite(value) && value >= 0 && value <= 100 ? value : undefined
}

/** Include retained sensors even if the most recent sample has no PWM data. */
export function pwmSensorNames(records: SystemStatsRecord[]): string[] {
	const sensors = new Set<string>()
	for (const record of records) {
		for (const sensor of Object.keys(record.stats?.pwm ?? {})) {
			if (pwmValue(record, sensor) !== undefined) {
				sensors.add(sensor)
			}
		}
	}
	return Array.from(sensors).sort()
}

/** Keep stable storage keys separate from labels, disambiguating only duplicate names. */
export function pwmSensorLabels(sensors: string[]): { sensor: string; label: string }[] {
	const parsed = sensors.map((sensor) => {
		const match = /^(.*_pwm[1-9]\d*)_([0-9a-f]{16})$/.exec(sensor)
		return { sensor, label: match?.[1] ?? sensor, suffix: match?.[2] }
	})
	const names = new Set(parsed.map(({ label }) => label))
	return parsed.map(({ sensor, label, suffix }) => {
		const peers = parsed.filter((other) => other.sensor !== sensor && other.label === label)
		if (!suffix || !peers.length) {
			return { sensor, label }
		}
		// An identity prefix is independent of discovery order and current output.
		for (let length = 4; length <= suffix.length; length++) {
			const prefix = suffix.slice(0, length)
			const candidate = `${label} (${prefix})`
			if (!names.has(candidate) && !peers.some((other) => other.suffix?.startsWith(prefix))) {
				return { sensor, label: candidate }
			}
		}
		// Preserve uniqueness even if a legacy name occupies every short alias.
		return { sensor, label: sensor }
	})
}
