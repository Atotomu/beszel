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
