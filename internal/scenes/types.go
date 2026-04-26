package scenes

import "time"

type Scene struct {
	Name      string        `json:"name"`
	CreatedAt time.Time     `json:"createdAt"`
	Groups    []SceneGroup  `json:"groups"`
	Devices   []SceneDevice `json:"devices"`
}

type SceneGroup struct {
	ID              string   `json:"id,omitempty"`
	CoordinatorUUID string   `json:"coordinatorUUID"`
	CoordinatorName string   `json:"coordinatorName,omitempty"`
	MemberUUIDs     []string `json:"memberUUIDs"`
}

type SceneDevice struct {
	UUID   string `json:"uuid"`
	Name   string `json:"name,omitempty"`
	IP     string `json:"ip,omitempty"`
	Volume int    `json:"volume"`
	Mute   bool   `json:"mute"`
}

type SceneMeta struct {
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
}
