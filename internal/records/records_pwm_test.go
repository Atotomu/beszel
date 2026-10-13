package records_test

import (
	"testing"

	"github.com/henrygd/beszel/internal/entities/system"
	"github.com/henrygd/beszel/internal/records"
	"github.com/stretchr/testify/assert"
)

func TestPWMAverageAvailableSamples(t *testing.T) {
	result := records.AverageSystemStatsSlice([]system.Stats{
		{PWM: map[string]float64{"pwm1": 0, "pwm2": 31.37, "stopped": 0}},
		{PWM: map[string]float64{"pwm1": 100, "pwm2": 50, "stopped": 0}},
		{PWM: map[string]float64{"pwm1": 50}},
		{PWM: map[string]float64{}},
		{},
	})
	assert.Equal(t, map[string]float64{"pwm1": 50, "pwm2": 40.69, "stopped": 0}, result.PWM)
}

func TestPWMAverageUnavailable(t *testing.T) {
	for _, input := range [][]system.Stats{nil, {}, {{}, {PWM: map[string]float64{}}}} {
		assert.Nil(t, records.AverageSystemStatsSlice(input).PWM)
	}
}

func TestPWMAverageUsesNativeWindowWeights(t *testing.T) {
	// Like the other native metrics, each available shorter window contributes
	// equally to a longer window, even if it has fewer underlying readings.
	first := records.AverageSystemStatsSlice([]system.Stats{
		{PWM: map[string]float64{"pwm1": 0}},
		{},
	})
	second := records.AverageSystemStatsSlice([]system.Stats{
		{PWM: map[string]float64{"pwm1": 100}},
		{PWM: map[string]float64{"pwm1": 100}},
	})
	result := records.AverageSystemStatsSlice([]system.Stats{first, second, {}})
	assert.Equal(t, map[string]float64{"pwm1": 50}, result.PWM)
}
