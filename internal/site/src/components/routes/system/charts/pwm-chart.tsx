import { t } from "@lingui/core/macro"
import { useStore } from "@nanostores/react"
import { atom } from "nanostores"
import { useMemo } from "react"
import LineChartDefault from "@/components/charts/line-chart"
import { pwmSensorNames, pwmValue } from "@/lib/pwm"
import { cn, decimalString, toFixedFloat } from "@/lib/utils"
import type { ChartData, SystemStatsRecord } from "@/types"
import { ChartCard, FilterBar } from "../chart-card"

const $pwmFilter = atom("")

export function PWMChart({ chartData, grid, dataEmpty }: { chartData: ChartData; grid: boolean; dataEmpty: boolean }) {
	const filter = useStore($pwmFilter)
	const sensorNamesKey = JSON.stringify(pwmSensorNames(chartData.systemStats))
	const dataPoints = useMemo(() => {
		const sensors: string[] = JSON.parse(sensorNamesKey)
		const filterTerms = filter.toLowerCase().split(" ").filter(Boolean)
		return sensors.map((sensor, index) => {
			const filtered = filterTerms.length > 0 && !filterTerms.some((term) => sensor.toLowerCase().includes(term))
			return {
				label: sensor,
				dataKey: (record: SystemStatsRecord) => pwmValue(record, sensor),
				color: `hsl(${((index * 360) / sensors.length) % 360}, 60%, 55%)`,
				strokeOpacity: filtered ? 0.1 : 1,
				activeDot: !filtered,
			}
		})
	}, [sensorNamesKey, filter])

	if (!dataPoints.length) {
		return null
	}

	const legend = dataPoints.length < 12

	return (
		<div className={cn("odd:last-of-type:col-span-full", { "col-span-full": !grid })}>
			<ChartCard
				empty={dataEmpty}
				grid={grid}
				title={t`PWM output`}
				description={t`Fan control output (%)`}
				cornerEl={<FilterBar store={$pwmFilter} />}
				legend={legend}
			>
				<LineChartDefault
					chartData={chartData}
					itemSorter={(a, b) => b.value - a.value}
					domain={[0, 100]}
					legend={legend}
					tickFormatter={(value) => `${toFixedFloat(value, 0)}%`}
					contentFormatter={(item) => `${decimalString(item.value, 1)}%`}
					dataPoints={dataPoints}
					filter={filter}
				/>
			</ChartCard>
		</div>
	)
}
