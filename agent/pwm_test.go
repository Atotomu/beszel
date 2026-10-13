//go:build testing

package agent

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/henrygd/beszel/internal/entities/system"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadHwmonPWM(t *testing.T) {
	root := t.TempDir()
	chip := filepath.Join(root, "hwmon0")
	writeFile(t, filepath.Join(chip, "name"), "pwmfan\n")
	for name, value := range map[string]string{
		"pwm1": "0\n", "pwm2": "255\n", "pwm3": "80\n", "pwm4": "128\n",
		"pwm5": "-1\n", "pwm6": "256\n", "pwm7": "invalid\n", "pwm8": "80.5\n",
		"pwm10": "\n", "pwm11": "18446744073709551616\n",
		"pwm1_enable": "1\n", "pwm1_freq": "25\n", "pwm0": "255\n",
	} {
		writeFile(t, filepath.Join(chip, name), value)
	}
	writeFile(t, filepath.Join(chip, "pwm4_label"), "case\n")
	// A directory cannot be read as a numeric file, even when tests run as root.
	require.NoError(t, os.Mkdir(filepath.Join(chip, "pwm9"), 0o755))
	// There is deliberately no fan*_input: PWM does not depend on a tachometer.
	sensors, err := discoverHwmonPWM(root)
	require.NoError(t, err)
	assert.Equal(t, map[string]float64{
		pwmKeyForPath(t, sensors, filepath.Join(chip, "pwm1")): 0,
		pwmKeyForPath(t, sensors, filepath.Join(chip, "pwm2")): 100,
		pwmKeyForPath(t, sensors, filepath.Join(chip, "pwm3")): 31.37,
		pwmKeyForPath(t, sensors, filepath.Join(chip, "pwm4")): 50.2,
	}, readPWMSensors(sensors, false))
}

func TestReadHwmonPWMLegacyParent(t *testing.T) {
	root := t.TempDir()
	device := filepath.Join(t.TempDir(), "platform-fan")
	writeFile(t, filepath.Join(device, "name"), "legacy\n")
	writeFile(t, filepath.Join(device, "pwm1"), "255\n")
	chip := filepath.Join(root, "hwmon1")
	require.NoError(t, os.Mkdir(chip, 0o755))
	require.NoError(t, os.Symlink(device, filepath.Join(chip, "device")))
	sensors, err := discoverHwmonPWM(root)
	require.NoError(t, err)
	key := pwmKeyForPath(t, sensors, filepath.Join(chip, "device", "pwm1"))
	assert.Regexp(t, `^legacy_pwm1_[a-f0-9]{16}$`, key)
	assert.Equal(t, map[string]float64{key: 100}, readPWMSensors(sensors, false))
}

func TestPWMDiscoveryMissingAndEmpty(t *testing.T) {
	_, err := discoverHwmonPWM(filepath.Join(t.TempDir(), "missing"))
	assert.Error(t, err)
	sensors, err := discoverHwmonPWM(t.TempDir())
	require.NoError(t, err)
	assert.Empty(t, readPWMSensors(sensors, false))
}

func TestPWMDiscoveryCache(t *testing.T) {
	root := t.TempDir()
	input := filepath.Join(root, "hwmon0", "pwm1")
	writeFile(t, filepath.Join(root, "hwmon0", "name"), "chip\n")
	writeFile(t, input, "80\n")
	getSensors := newPWMSensorCache(root)
	sensors, err := getSensors()
	require.NoError(t, err)
	key := pwmKeyForPath(t, sensors, input)
	assert.Equal(t, map[string]float64{key: 31.37}, readPWMSensors(sensors, false))
	writeFile(t, input, "128\n")
	writeFile(t, filepath.Join(root, "hwmon0", "pwm1_label"), "renamed\n")
	sensors, err = getSensors()
	require.NoError(t, err)
	assert.Equal(t, map[string]float64{key: 50.2}, readPWMSensors(sensors, false))
	require.NoError(t, os.Remove(input))
	assert.Empty(t, readPWMSensors(sensors, false), "unavailable readings must not become 0%% or reuse the old value")
}

func TestPWMNamesSurviveHwmonRenumbering(t *testing.T) {
	root := t.TempDir()
	devices := t.TempDir()
	// Two identical chips and two identically labeled channels must not merge.
	for i, device := range []string{"platform-a", "platform-b"} {
		dir := filepath.Join(devices, device, "hwmon", "hwmon0")
		writeFile(t, filepath.Join(dir, "name"), "chip\n")
		writeFile(t, filepath.Join(dir, "pwm1"), "80\n")
		writeFile(t, filepath.Join(dir, "pwm2"), "128\n")
		writeFile(t, filepath.Join(dir, "pwm1_label"), "case\n")
		writeFile(t, filepath.Join(dir, "pwm2_label"), "case\n")
		require.NoError(t, os.Symlink(dir, filepath.Join(root, []string{"hwmon0", "hwmon1"}[i])))
	}
	sensors, err := discoverHwmonPWM(root)
	require.NoError(t, err)
	before := readPWMSensors(sensors, false)
	require.Len(t, before, 4)
	for key := range before {
		assert.Regexp(t, `^chip_pwm[12]_[a-f0-9]{16}$`, key)
	}
	require.NoError(t, os.Remove(filepath.Join(root, "hwmon0")))
	require.NoError(t, os.Remove(filepath.Join(root, "hwmon1")))
	for i, device := range []string{"platform-a", "platform-b"} {
		parent := filepath.Join(devices, device, "hwmon")
		require.NoError(t, os.Rename(filepath.Join(parent, "hwmon0"), filepath.Join(parent, "hwmon7")))
		require.NoError(t, os.Symlink(filepath.Join(parent, "hwmon7"), filepath.Join(root, []string{"hwmon9", "hwmon3"}[i])))
	}
	sensors, err = discoverHwmonPWM(root)
	require.NoError(t, err)
	assert.Equal(t, before, readPWMSensors(sensors, false))
}

func TestPWMIdentityIndependentOfOtherDevices(t *testing.T) {
	root := t.TempDir()
	devices := t.TempDir()
	firstDevice := filepath.Join(devices, "platform-a", "hwmon", "hwmon0")
	secondDevice := filepath.Join(devices, "platform-b", "hwmon", "hwmon1")
	for _, device := range []string{firstDevice, secondDevice} {
		writeFile(t, filepath.Join(device, "name"), "pwmfan\n")
		writeFile(t, filepath.Join(device, "pwm1"), "80\n")
	}
	firstLink := filepath.Join(root, "hwmon0")
	secondLink := filepath.Join(root, "hwmon1")
	require.NoError(t, os.Symlink(firstDevice, firstLink))

	discover := func() []pwmSensor {
		t.Helper()
		// Each discovery represents a fresh agent process, not a retained cache.
		sensors, err := discoverHwmonPWM(root)
		require.NoError(t, err)
		return sensors
	}
	before := discover()
	require.Len(t, before, 1)
	key := pwmKeyForPath(t, before, filepath.Join(firstLink, "pwm1"))
	assert.Regexp(t, `^pwmfan_pwm1_[a-f0-9]{16}$`, key)

	require.NoError(t, os.Symlink(secondDevice, secondLink))
	withSecond := discover()
	require.Len(t, withSecond, 2)
	assert.Equal(t, key, pwmKeyForPath(t, withSecond, filepath.Join(firstLink, "pwm1")))
	assert.NotEqual(t, key, pwmKeyForPath(t, withSecond, filepath.Join(secondLink, "pwm1")))

	require.NoError(t, os.Remove(secondLink))
	// Labels are presentation metadata and must not change the series identity.
	writeFile(t, filepath.Join(firstDevice, "pwm1_label"), "new label\n")
	after := discover()
	require.Len(t, after, 1)
	assert.Equal(t, key, pwmKeyForPath(t, after, filepath.Join(firstLink, "pwm1")))
}

func TestPWMSkipGPU(t *testing.T) {
	root := t.TempDir()
	for _, chip := range []string{"xe", "nct6798"} {
		writeFile(t, filepath.Join(root, chip, "name"), chip)
		writeFile(t, filepath.Join(root, chip, "pwm1"), "255\n")
	}
	sensors, err := discoverHwmonPWM(root)
	require.NoError(t, err)
	assert.Len(t, readPWMSensors(sensors, false), 2)
	key := pwmKeyForPath(t, sensors, filepath.Join(root, "nct6798", "pwm1"))
	assert.Equal(t, map[string]float64{key: 100}, readPWMSensors(sensors, true))
	assert.Len(t, sensors, 2, "GPU filtering must not mutate the shared discovery cache")
}

func TestUpdatePWMPlatformSupport(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "hwmon0", "name"), "pwmfan\n")
	writeFile(t, filepath.Join(root, "hwmon0", "pwm1"), "0\n")
	previous := getPWMSensors
	getPWMSensors = newPWMSensorCache(root)
	t.Cleanup(func() { getPWMSensors = previous })
	var stats system.Stats
	(&Agent{}).updatePWM(&stats)
	if hwmonRoot == "" {
		assert.Nil(t, stats.PWM, "non-Linux platforms must not collect sysfs PWM")
	} else {
		sensors, err := getPWMSensors()
		require.NoError(t, err)
		key := pwmKeyForPath(t, sensors, filepath.Join(root, "hwmon0", "pwm1"))
		assert.Equal(t, map[string]float64{key: 0}, stats.PWM)
	}
}

func pwmKeyForPath(t *testing.T, sensors []pwmSensor, path string) string {
	t.Helper()
	for _, sensor := range sensors {
		if sensor.path == path {
			return sensor.key
		}
	}
	t.Fatalf("no PWM sensor for %s", path)
	return ""
}
