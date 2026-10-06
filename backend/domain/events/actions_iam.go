package events

// IAM actions (backend/application/iam). Pairing codes and tokens are
// secrets and never appear in events.
var (
	GroupCreated = defineAction("iam.group.created",
		"A permission group was created. data: name, description.")
	GroupUpdated = defineAction("iam.group.updated",
		"A permission group changed (changes: name, description).")
	GroupDeleted = defineAction("iam.group.deleted",
		"A permission group was deleted. data: name.")
	GroupMemberAdded = defineAction("iam.group_member.added",
		"A user joined a permission group. entity_id is the group. data: user_id.")
	GroupMemberRemoved = defineAction("iam.group_member.removed",
		"A user left a permission group. entity_id is the group. data: user_id.")
	GroupPermissionAssigned = defineAction("iam.group_permission.assigned",
		"A permission rule was granted to a group (re-granting an existing rule raises nothing). entity_id is the group. data: rule.")
	GroupPermissionRevoked = defineAction("iam.group_permission.revoked",
		"A permission rule was taken from a group. entity_id is the group. data: rule.")
	ScannerPairingCodeCreated = defineAction("iam.scanner_pairing_code.created",
		"A user asked to pair a scanner device. data: user_id, role, expires_at.")
	ScannerPairingCodeClaimed = defineAction("iam.scanner_pairing_code.claimed",
		"A scanner device claimed a pairing code and got a session acting as the pairing user (the actor). data: user_id, role.")
)
