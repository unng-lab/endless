package unit

import "log"

// SetExternalAPIDebugLogging enables or disables verbose logging for the external movement and
// fire order API. Visual debug runs turn this on so operators can correlate input or policy
// decisions with the exact manager command that was accepted or rejected.
func (m *Manager) SetExternalAPIDebugLogging(enabled bool) {
	if m == nil {
		return
	}

	m.debugExternalAPILogging = enabled
}

// debugExternalAPILogf centralizes the log prefix for manager-level command tracing so move and
// fire API calls stay easy to grep in one mixed runtime log stream.
func (m *Manager) debugExternalAPILogf(format string, args ...any) {
	if m == nil || !m.debugExternalAPILogging {
		return
	}

	log.Printf("[debug][external-api] "+format, args...)
}

// debugUnitRuntimeLogf emits lower-level order and movement traces from inside one mobile
// unit. Keeping it behind the same debug flag lets operators switch on both the API audit
// trail and the unit-internal handoff trace in one place during gameplay investigations.
func (m *Manager) debugUnitRuntimeLogf(format string, args ...any) {
	if m == nil || !m.debugExternalAPILogging {
		return
	}

	log.Printf("[debug][unit-runtime] "+format, args...)
}
