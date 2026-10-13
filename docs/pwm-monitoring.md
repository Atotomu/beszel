# PWM monitoring

This fork adds read-only Linux hwmon PWM duty-cycle monitoring. It does not
configure the fan, change its control mode, or require an RPM sensor.

The agent discovers `pwmN` attributes below `/sys/class/hwmon` (including legacy
`device/` attributes) at startup and reports valid values from 0–255 as 0–100%.
`0` is a valid reading. Missing, unreadable, malformed, and out-of-range values
are omitted. Platforms without hwmon report no PWM data. The agent needs read
access to the host's hwmon files; no write access is required. `SKIP_GPU` also
skips GPU PWM readings. Restart the agent after changing the available sensors,
as discovery is cached like native fan discovery.

PWM describes the driver's control output, not measured fan speed. It cannot
prove that the fan is physically turning or be converted to RPM. Sensors use
the chip and PWM channel plus a physical-device identity suffix so identical
chips do not overwrite one another. Moving a device to a different physical
path can change its identity.

The hub stores `stats.pwm` in the existing `system_stats` collection and uses
the existing transport, subscriptions, retention, and aggregation schedule:

| View | Record interval |
| --- | --- |
| Live | Native live sampling |
| 1 hour | 1 minute |
| 12 hours | 10 minutes |
| 24 hours | 20 minutes |
| 1 week | 120 minutes |
| 30 days | 480 minutes |

Each aggregation averages the available shorter records for each PWM channel.
Missing channels do not count as zero; real zero readings do count. Longer
windows follow native equal-weight aggregation of shorter-window averages,
not a separate time-weighted or change-event history. They do not preserve
every intermediate output change. No database migration or new collection is
needed.

The PWM card appears in the default layout and the Core tab when the selected
window contains valid PWM data. It uses native `ChartCard`, `FilterBar`, and
`LineChartDefault`: the same curve, legend, hover cursor, tooltip, touch
behavior, theme, and responsive layout. Only the metric, 0–100% axis, and
percentage formatting differ. Missing samples remain gaps. RPM charts remain
independent. There are no PWM configuration controls, gear numbers, or event
dialogs.

The feature is isolated in the PWM collector and chart, with small additions
to system stats, averaging, and the two existing layouts. Future upstream
updates should retain those integration points and reserve a non-conflicting
CBOR key for PWM (currently `41`).
