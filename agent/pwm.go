package agent

import (
	"crypto/sha256"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/henrygd/beszel/agent/utils"
	"github.com/henrygd/beszel/internal/entities/system"
)

type pwmSensor struct {
	key, path, chip string
}

var getPWMSensors = newPWMSensorCache(hwmonRoot)

func newPWMSensorCache(root string) func() ([]pwmSensor, error) {
	return sync.OnceValues(func() ([]pwmSensor, error) {
		return discoverHwmonPWM(root)
	})
}

// updatePWM reports the hwmon duty cycle independently of fan RPM support.
// Reading pwmN does not change the fan's control mode or output.
func (a *Agent) updatePWM(systemStats *system.Stats) {
	if hwmonRoot == "" {
		return
	}
	sensors, err := getPWMSensors()
	if err != nil {
		slog.Debug("Error reading PWM", "err", err)
		return
	}
	if pwm := readPWMSensors(sensors, a.sensorConfig != nil && a.sensorConfig.skipGPU); len(pwm) > 0 {
		systemStats.PWM = pwm
	}
}

func discoverHwmonPWM(root string) ([]pwmSensor, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var sensors []pwmSensor
	for _, entry := range entries {
		chipDir := filepath.Join(root, entry.Name())
		sensorDir := chipDir
		inputs := pwmInputs(sensorDir)
		// Legacy drivers expose their attributes on the parent device.
		if len(inputs) == 0 {
			sensorDir = filepath.Join(chipDir, "device")
			inputs = pwmInputs(sensorDir)
		}
		chip := utils.ReadStringFile(filepath.Join(sensorDir, "name"))
		if chip == "" {
			chip = utils.ReadStringFile(filepath.Join(chipDir, "name"))
		}
		if chip == "" {
			chip = "hwmon"
		}
		identity := pwmDeviceIdentity(chipDir)
		for _, input := range inputs {
			base := filepath.Base(input)
			// Always include the physical identity: adding another identical chip
			// must not rename an existing history series after agent restart.
			digest := sha256.Sum256([]byte(identity + "/" + base))
			key := fmt.Sprintf("%s_%s_%x", chip, base, digest[:8])
			sensors = append(sensors, pwmSensor{key: key, path: input, chip: chip})
		}
	}
	return sensors, nil
}

// pwmInputs excludes control attributes such as pwm1_enable and pwm1_freq.
func pwmInputs(dir string) []string {
	paths, _ := filepath.Glob(filepath.Join(dir, "pwm[1-9]*"))
	inputs := paths[:0]
	for _, path := range paths {
		channel := strings.TrimPrefix(filepath.Base(path), "pwm")
		if strings.Trim(channel, "0123456789") == "" {
			inputs = append(inputs, path)
		}
	}
	return inputs
}

// Prefer the hardware parent (including for legacy hwmon layouts). Modern
// class links resolve to <physical-device>/hwmon/hwmonN; remove that volatile
// suffix so discovery order does not determine the sensor's identity.
func pwmDeviceIdentity(chipDir string) string {
	device, err := filepath.EvalSymlinks(filepath.Join(chipDir, "device"))
	if err != nil {
		device, err = filepath.EvalSymlinks(chipDir)
		if err != nil {
			return chipDir
		}
	}
	if filepath.Base(filepath.Dir(device)) == "hwmon" {
		return filepath.Dir(filepath.Dir(device))
	}
	return device
}

// hwmon PWM values are in [0, 255]. Missing and invalid samples are omitted;
// an actual zero remains a valid duty cycle, even if no fan*_input exists.
func readPWMSensors(sensors []pwmSensor, skipGPU bool) map[string]float64 {
	pwm := make(map[string]float64, len(sensors))
	for _, sensor := range sensors {
		// Like fan RPM reads, some GPU sensor reads can wake an idle device.
		if skipGPU && isGpuChipName(sensor.chip) {
			continue
		}
		if raw, ok := utils.ReadUintFile(sensor.path); ok && raw <= 255 {
			pwm[sensor.key] = utils.TwoDecimals(float64(raw) * 100 / 255)
		}
	}
	return pwm
}
