package hypervisor

import (
	"log"
	"time"

	"internal/blueteam"
)

// TriggerHealing is called by the Red Team on fault detection.
func TriggerHealing(healer *blueteam.Healer, faultReason string, isCritical bool) {
	const healingLimit = 60 * time.Second

	var duration time.Duration

	if isCritical {
		duration = healer.ExecuteHardReversion(faultReason)
	} else {
		// Example: initiate soft patch to adjust threshold slightly
		duration = healer.ExecuteSoftPatch(faultReason, 3.2)
	}

	// A-3 Compliance Check
	if duration > healingLimit {
		// CRITICAL FAILURE: The system could not self-correct fast enough.
		log.Fatalf("[SCGO A-3 FAILURE] Time-to-Heal exceeded %s limit! Duration: %s", healingLimit, duration)
	}

	// Log the successful Time-to-Heal for SBOH
	log.Printf("[SCGO A-3 SUCCESS] Healing complete and compliant. Duration: %.2fs", duration.Seconds())
}
