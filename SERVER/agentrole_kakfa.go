package server

import()


type AgentRole struct {
	ID        int   `json:"id"`
	Role      string `json:"role"` // "sender" ou "reflector"
	TargetIDs []int  `json:"target_ids,omitempty"`
}


func BuildAgentRoles(sourceID int, targetIDs []int) []AgentRole {
	var roles []AgentRole

	// Ajouter le sender
	roles = append(roles, AgentRole{
		ID:        sourceID,
		Role:      "sender",
		TargetIDs: targetIDs,
	})

	// Ajouter tous les reflectors
	for _, targetID := range targetIDs {
		roles = append(roles, AgentRole{
			ID:   targetID,
			Role: "reflector",
		})
	}

	return roles
}

