import { expect, test } from "bun:test"
import type { SystemStatsRecord } from "@/types"
import { pwmSensorLabels, pwmSensorNames, pwmValue } from "./pwm"

const record = (pwm?: Record<string, number>) => ({ stats: { pwm } }) as SystemStatsRecord

test("PWM keeps zero and fractional percentages without filling missing samples", () => {
	expect(pwmValue(record({ pwm1: 0 }), "pwm1")).toBe(0)
	expect(pwmValue(record({ pwm1: 31.37 }), "pwm1")).toBe(31.37)
	expect(pwmValue(record({ pwm1: 100 }), "pwm1")).toBe(100)
	expect(pwmValue(record(), "pwm1")).toBeUndefined()
	expect(pwmValue(record({ pwm2: 50 }), "pwm1")).toBeUndefined()
})

test("PWM rejects malformed percentages instead of plotting them as a valid output", () => {
	for (const value of [-1, 100.1, Number.NaN, Number.POSITIVE_INFINITY]) {
		expect(pwmValue(record({ pwm1: value }), "pwm1")).toBeUndefined()
	}
	const pwm = { pwm1: 50 }
	const malformed = record(pwm)
	Reflect.set(pwm, "pwm1", null)
	expect(pwmValue(malformed, "pwm1")).toBeUndefined()
	Reflect.set(pwm, "pwm1", "50")
	expect(pwmValue(malformed, "pwm1")).toBeUndefined()
})

test("PWM chart discovery includes historical and zero-only sensors independently of RPM", () => {
	expect(pwmSensorNames([record(), record({})])).toEqual([])
	expect(pwmSensorNames([record({ invalid: -1 })])).toEqual([])
	expect(pwmSensorNames([record({ pwm2: 0 }), record({ pwm1: 50 }), record()])).toEqual(["pwm1", "pwm2"])
})

test("PWM sensor ordering stays stable as outputs change", () => {
	expect(pwmSensorNames([record({ pwm2: 90, pwm1: 10 })])).toEqual(["pwm1", "pwm2"])
	expect(pwmSensorNames([record({ pwm2: 10, pwm1: 90 })])).toEqual(["pwm1", "pwm2"])
})

test("PWM labels hide only collector identity suffixes without changing data keys", () => {
	const sensor = "pwmfan_pwm1_128af9bf1a968f40"
	expect(pwmSensorLabels([sensor, "chassis_pwm2", "custom_128af9bf1a968f40"])).toEqual([
		{ sensor, label: "pwmfan_pwm1" },
		{ sensor: "chassis_pwm2", label: "chassis_pwm2" },
		{ sensor: "custom_128af9bf1a968f40", label: "custom_128af9bf1a968f40" },
	])
	const sample = record({ [sensor]: 31.37 })
	expect(pwmValue(sample, pwmSensorLabels([sensor])[0].sensor)).toBe(31.37)
})

test("Duplicate PWM labels use unique identity prefixes independent of ordering", () => {
	const sensors = ["pwmfan_pwm1_128af9bf1a968f40", "pwmfan_pwm1_128bf9bf1a968f40", "pwmfan_pwm1_5678f9bf1a968f40"]
	const expected = [
		{ sensor: sensors[0], label: "pwmfan_pwm1 (128a)" },
		{ sensor: sensors[1], label: "pwmfan_pwm1 (128b)" },
		{ sensor: sensors[2], label: "pwmfan_pwm1 (5678)" },
	]
	expect(pwmSensorLabels(sensors)).toEqual(expected)
	expect(pwmSensorLabels([...sensors].reverse())).toEqual([...expected].reverse())
})

test("PWM labels extend colliding prefixes and preserve legacy names", () => {
	const sensors = ["pwmfan_pwm1_128af9bf1a968f40", "pwmfan_pwm1_128ab9bf1a968f40", "pwmfan_pwm1", "pwmfan_pwm1 (128af)"]
	expect(pwmSensorLabels(sensors).map(({ label }) => label)).toEqual([
		"pwmfan_pwm1 (128af9)",
		"pwmfan_pwm1 (128ab)",
		"pwmfan_pwm1",
		"pwmfan_pwm1 (128af)",
	])
})
