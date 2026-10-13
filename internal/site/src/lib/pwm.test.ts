import { expect, test } from "bun:test"
import type { SystemStatsRecord } from "@/types"
import { pwmSensorNames, pwmValue } from "./pwm"

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
